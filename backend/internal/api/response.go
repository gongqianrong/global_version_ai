package api

import (
	"encoding/json"
	"net/http"

	"github.com/rakutao/collection-gateway/internal/i18n"
)

// APIResponse is the standard JSON envelope for all API responses.
type APIResponse struct {
	Code      int         `json:"code"`
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

// contextKey is an unexported type for context keys in this package.
type contextKey string

// requestIDKey is the context key for the request ID.
const requestIDKey contextKey = "request_id"

// Success writes a successful JSON response with code 0.
func Success(w http.ResponseWriter, r *http.Request, data interface{}) {
	writeJSON(w, http.StatusOK, APIResponse{
		Code:      0,
		Data:      data,
		RequestID: getRequestID(r),
	})
}

// ErrorWithCode writes an error JSON response with the given HTTP status and business error code.
func ErrorWithCode(w http.ResponseWriter, r *http.Request, httpStatus, code int, message string) {
	lang := i18n.FromContext(r.Context())
	translated := i18n.ErrorMessage(code, lang)
	writeJSON(w, httpStatus, APIResponse{
		Code:      code,
		Message:   translated,
		RequestID: getRequestID(r),
	})
}

// writeJSON marshals v to JSON and writes it to w with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// getRequestID extracts the request ID from the request context.
func getRequestID(r *http.Request) string {
	if id, ok := r.Context().Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}
