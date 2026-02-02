package api

import (
	"net/http"
)

type ErrorResponse struct {
	Error APIError `json:"error"`
}

type APIError struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
}

func WriteError(w http.ResponseWriter, status int, code, message, requestID string, details map[string]any) {
	WriteJSON(w, status, ErrorResponse{Error: APIError{Code: code, Message: message, Details: details, RequestID: requestID}})
}

// Convenience helpers
func BadRequest(w http.ResponseWriter, code, message, requestID string, details map[string]any) {
	WriteError(w, http.StatusBadRequest, code, message, requestID, details)
}

func Unauthorized(w http.ResponseWriter, code, message, requestID string) {
	WriteError(w, http.StatusUnauthorized, code, message, requestID, nil)
}

func Forbidden(w http.ResponseWriter, code, message, requestID string) {
	WriteError(w, http.StatusForbidden, code, message, requestID, nil)
}

func NotFound(w http.ResponseWriter, code, message, requestID string) {
	WriteError(w, http.StatusNotFound, code, message, requestID, nil)
}

func Conflict(w http.ResponseWriter, code, message, requestID string, details map[string]any) {
	WriteError(w, http.StatusConflict, code, message, requestID, details)
}

func RateLimited(w http.ResponseWriter, code, message, requestID string, details map[string]any) {
	WriteError(w, http.StatusTooManyRequests, code, message, requestID, details)
}

func Internal(w http.ResponseWriter, requestID string) {
	WriteError(w, http.StatusInternalServerError, "INTERNAL", "Internal server error", requestID, nil)
}
