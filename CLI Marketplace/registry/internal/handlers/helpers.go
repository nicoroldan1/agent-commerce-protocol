package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/nicroldan/ans/shared/ace"
)

// writeJSON writes v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// decodeJSON decodes the request body into v.
func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// writeError writes an ace.ErrorResponse with the given status code.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ace.ErrorResponse{
		Error: message,
		Code:  code,
	})
}
