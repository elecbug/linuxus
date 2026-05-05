package http_helper

import (
	"encoding/json"
	"net/http"
)

// WriteJSONViaHTTP writes a JSON response with the given HTTP status code.
func WriteJSONViaHTTP(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
