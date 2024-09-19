package handlehttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type (
	// TargetFunc is a generic function type that executes bussiness logic
	TargetFunc[in validator, out any] func(context.Context, in) (out out, err error)
	VoidTargetFunc[out any]           func(context.Context) (out out, err error)
	// validator is an object that can be validated and decoded.
	validator interface {
		// Valid checks the object and returns any
		// problems. If len(problems) == 0 then
		// the object is valid.
		Valid(context.Context) (problems map[string]string)
		// Decode decodes the query parameters from the request into the object.
		// This overrides any values obtained from the body.
		Decode(context.Context, *http.Request) (err error)
	}
	Logger interface {
		Errorf(format string, args ...interface{})
		Infof(format string, args ...interface{})
		Warnf(format string, args ...interface{})
	}
	statusError interface {
		error
		Status() int
	}
	err struct {
		msg    string
		status int
	}
)

func Error(msg string, status int) statusError {
	return err{msg: msg, status: status}
}

func (e err) Error() string {
	return e.msg
}

func (e err) Status() int {
	return e.status
}

func formatProblems(problems map[string]string) string {
	msg := ""
	for k, v := range problems {
		msg += fmt.Sprintf("%s: %s\n", k, v)
	}

	return msg
}

func HandleVoid[out any](log Logger, f VoidTargetFunc[out]) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Call out to target function
		out, err := f(r.Context())
		if err != nil {
			badRequest(log, err.Error(), w, http.StatusBadRequest)
			return
		}

		// Format and write response
		respond(http.StatusOK, w, log, out)
	})
}

// HandleValid is a generic handler for http requests
func HandleValid[in validator, out any](log Logger, f TargetFunc[in, out]) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode body
		var input in

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil && err.Error() != "EOF" /* ignore empty body */ {
			badRequest(log, fmt.Sprintf("handler failed to decode body: %v", err), w, http.StatusBadRequest)
			return
		} else if err != nil && err.Error() == "EOF" { //initialize empty body if no body exists
			byt := []byte(`{}`)
			if err := json.Unmarshal(byt, &input); err != nil {
				badRequest(log, fmt.Sprintf("failed to encode body: %v", err), w, http.StatusBadRequest)
				return
			}
		}

		// Decode query parameters
		if err := input.Decode(r.Context(), r); err != nil {
			badRequest(log, fmt.Sprintf("handler failed to decode query: %v", err), w, http.StatusBadRequest)
			return
		}

		// Validate request
		if problems := input.Valid(r.Context()); len(problems) > 0 {
			badRequest(log, fmt.Sprintf("handler failed to validate request: %s", formatProblems(problems)), w, http.StatusBadRequest)
			return
		}

		// Call out to target function
		out, err := f(r.Context(), input)
		if httpERR, ok := err.(statusError); ok {
			badRequest(log, err.Error(), w, httpERR.Status())
			return
		}

		if err != nil {
			badRequest(log, err.Error(), w, http.StatusBadRequest)
			return
		}

		// Format and write response
		respond(http.StatusOK, w, log, out)
	})
}

func badRequest(log Logger, msg string, w http.ResponseWriter, status int) {
	log.Warnf(msg)
	respond(status, w, log, map[string]string{"error": msg})
}

func respond(status int, w http.ResponseWriter, logger_ Logger, res interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(res)

	if err != nil {
		logger_.Errorf("failed to encode response: %v", err)
	}
}
