package httpx

import (
	"encoding/json"
	"net/http"
)

// JSON sends a JSON response with status code and payload object.
func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// Error is a convenience helper for standard {"error":"..."} responses.
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}

