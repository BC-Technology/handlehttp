package handlehttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type (
	// targetFunc is a generic function type that executes bussiness logic
	targetFunc[in validator, out any] func(context.Context, in, ...interface{}) (out, error)
	// validator is an object that can be validated and decoded.
	validator interface {
		// Valid checks the object and returns any
		// problems. If len(problems) == 0 then
		// the object is valid.
		Valid(context.Context) (problems map[string]string)
		// Decode decodes the query parameters from the request into the object.
		// This overrides any values obtained from the body.
		Decode(context.Context, *http.Request) error
	}
	Logger interface {
		Errorf(format string, args ...interface{})
		Infof(format string, args ...interface{})
		Warnf(format string, args ...interface{})
	}
)

func formatProblems(problems map[string]string) string {
	msg := ""
	for k, v := range problems {
		msg += fmt.Sprintf("%s: %s\n", k, v)
	}

	return msg
}

// Handle is a generic handler for http requests
func Handle[in validator, out any](log Logger, f targetFunc[in, out], args ...interface{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode body
		var input in

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil && err.Error() != "EOF" /* ignore empty body */ {
			badRequest(log, fmt.Sprintf("handler failed to decode body: %v", err), w)
			return
		} else if err.Error() == "EOF" { //initialize empty body if no body exists
			byt := []byte(`{}`)
			if err := json.Unmarshal(byt, &input); err != nil {
				badRequest(log, fmt.Sprintf("failed to encode body: %v", err), w)
				return
			}
		}

		// Decode query parameters
		if err := input.Decode(r.Context(), r); err != nil {
			badRequest(log, fmt.Sprintf("handler failed to decode query: %v", err), w)
			return
		}

		// Validate request
		if problems := input.Valid(r.Context()); len(problems) > 0 {
			badRequest(log, fmt.Sprintf("handler failed to validate request: %s", formatProblems(problems)), w)
			return
		}

		// Call out to target function
		out, err := f(r.Context(), input, args...)
		if err != nil {
			badRequest(log, err.Error(), w)
			return
		}

		// Format and write response
		respond(http.StatusOK, w, log, out)
	})
}

func badRequest(log Logger, msg string, w http.ResponseWriter) {
	log.Warnf(msg)
	respond(http.StatusBadRequest, w, log, map[string]string{"error": msg})
}

func respond(status int, w http.ResponseWriter, logger_ Logger, res interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(res)

	if err != nil {
		logger_.Errorf("failed to encode response: %v", err)
	}
}
