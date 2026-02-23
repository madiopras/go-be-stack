package handlers

import (
	"betest/internal/response"
	"net/http"
)

// Health returns API health status for FE/load balancer checks.
func Health(w http.ResponseWriter, r *http.Request) {
	response.SendSuccess(w, http.StatusOK, "ok", map[string]string{"status": "up"})
}
