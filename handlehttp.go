package handlehttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type (
	// targetFunc is a generic function type that executes bussiness logic
	targetFunc[in any, out any] func(context.Context, in) (out, error)
	// validator is an object that can be validated.
	validator interface {
		// Valid checks the object and returns any
		// problems. If len(problems) == 0 then
		// the object is valid.
		Valid(context.Context) (problems map[string]string)
	}
	logger interface {
		Errorf(format string, args ...interface{})
		Infof(format string, args ...interface{})
		Warnf(format string, args ...interface{})
	}
)

func decodeValid[T validator](r *http.Request) (T, map[string]string, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, nil, fmt.Errorf("decode json: %w", err)
	}

	if problems := v.Valid(r.Context()); len(problems) > 0 {
		return v, problems, fmt.Errorf("invalid %T: %d problems", v, len(problems))
	}

	return v, nil, nil
}

func formatProblems(problems map[string]string) string {
	msg := ""
	for k, v := range problems {
		msg += fmt.Sprintf("%s: %s\n", k, v)
	}

	return msg
}

// HandleValid is a generic handler for http requests
func HandleValid[in validator, out any](log logger, f targetFunc[in, out]) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode and validate request
		in, problems, err := decodeValid[in](r)
		if err != nil {
			msg := ""
			if len(problems) > 0 {
				msg = fmt.Sprintf("failed to decode request: %v, problems: %s", err, formatProblems(problems))
			} else {
				msg = fmt.Sprintf("failed to decode request: %v", err)
			}

			log.Warnf("request validation: %s", msg)

			respond(http.StatusBadRequest, w, log, map[string]string{"error": msg})

			return
		}

		// Call out to target function
		out, err := f(r.Context(), in)
		if err != nil {
			respond(http.StatusBadRequest, w, log, map[string]string{"error": err.Error()})
			return
		}

		// Format and write response
		respond(http.StatusOK, w, log, out)
	})
}

func respond(status int, w http.ResponseWriter, logger_ logger, res interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(res)

	if err != nil {
		logger_.Errorf("failed to encode response: %v", err)
	}
}
