// Package httpx contains small helpers shared by all HTTP handlers.
package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const maxBodyBytes = 1 << 20 // 1 MiB

// ErrorBody is the JSON envelope for every error response.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail describes a single error in a machine- and human-readable way.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSON serialises value as the response body with the given status code.
func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	// The status line is already sent; an encoding failure can only be a broken connection.
	_ = json.NewEncoder(w).Encode(value)
}

// WriteError writes the standard error envelope.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorBody{Error: ErrorDetail{Code: code, Message: message}})
}

// DecodeJSON reads a single JSON object from the request body into dst.
// Unknown fields and trailing data are rejected.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}
	if decoder.More() {
		return errors.New("invalid request body: unexpected trailing data")
	}
	return nil
}
