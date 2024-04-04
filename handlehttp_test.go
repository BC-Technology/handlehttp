package handlehttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/BC-Technology/handlehttp"
	"github.com/stretchr/testify/require"
)

type (
	// MockLogger is a mock implementation of the Logger interface
	MockLogger struct{}
	// MockValidator is a mock implementation of the validator interface
	MockValidator struct {
		Message string `json:"message"`
		ID      int    `json:"id"`
	}
)

// Valid checks the object and returns any problems. If len(problems) == 0 then the object is valid.
func (v MockValidator) Valid(ctx context.Context) map[string]string { return nil }

// Decode decodes the query parameters from the request into the object. This overrides any values obtained from the body.
func (v *MockValidator) Decode(ctx context.Context, r *http.Request) error {
	params := r.URL.Query()
	id := params.Get("id")

	// parse the id
	var err error
	v.ID, err = strconv.Atoi(id)
	if err != nil {
		return fmt.Errorf("parse id: %w", err)
	}

	return nil
}

// Errorf logs an error message
func (l *MockLogger) Errorf(format string, args ...interface{}) {}

// Infof logs an info message
func (l *MockLogger) Infof(format string, args ...interface{}) {}

// Warnf logs a warning message
func (l *MockLogger) Warnf(format string, args ...interface{}) {}

func TestHandleValid(t *testing.T) {
	// Create a mock logger
	mockLogger := &MockLogger{}
	type output struct {
		Message string `json:"message"`
		ID      int    `json:"id"`
	}

	expectedMessage := "Hello, World!"
	expectedID := 1
	expectedOutput := output{Message: expectedMessage, ID: expectedID}

	// Create a mock target function
	mockTargetFunc := func(ctx context.Context, in *MockValidator, args ...interface{}) (out any, err error) {
		// Return a mock output
		return output{Message: in.Message, ID: in.ID}, nil
	}

	// create a mock body
	mockBody, err := json.Marshal(MockValidator{Message: expectedMessage})
	require.NoError(t, err)

	// Create a mock request
	mockRequest := httptest.NewRequest(
		"GET",
		fmt.Sprintf("/?id=%d", expectedID),
		bytes.NewReader(mockBody),
	)

	// Create a mock response writer
	mockResponseWriter := httptest.NewRecorder()

	// Call the HandleValid function
	handler := handlehttp.Handle(mockLogger, mockTargetFunc)
	handler.ServeHTTP(mockResponseWriter, mockRequest)

	// Check the response status code
	require.Equal(t, http.StatusOK, mockResponseWriter.Code)

	// Check the response body
	actualResponseBody := output{}

	err = json.NewDecoder(mockResponseWriter.Body).Decode(&actualResponseBody)
	require.NoError(t, err)

	require.Equal(t, expectedOutput, actualResponseBody)
}
