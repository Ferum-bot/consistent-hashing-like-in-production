package response

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type errorResponse struct {
	Error string `json:"error"`
}

func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Warn("write response", "err", err)
	}
}

func Error(w http.ResponseWriter, status int, err error) {
	if status >= http.StatusInternalServerError {
		slog.Error("request failed", "status", status, "err", err)
	}

	JSON(w, status, errorResponse{Error: err.Error()})
}
