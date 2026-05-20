package health

import (
	"encoding/json"
	"net/http"
)

type CheckFunc func() error

type Response struct {
	Status         string `json:"status"`
	Database       string `json:"database"`
	DatabaseDriver string `json:"databaseDriver"`
}

func NewHandler(databaseDriver string, checkDatabase CheckFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := Response{
			Status:         "ok",
			Database:       "ok",
			DatabaseDriver: databaseDriver,
		}

		status := http.StatusOK
		if err := checkDatabase(); err != nil {
			response.Status = "degraded"
			response.Database = "error"
			status = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(response)
	})
}
