// Package handler contains health handlers
package handler

import (
	"encoding/json"
	"net/http"
)

// HealthHandler struct
type HealthHandler struct{}

// NewHealthHandler function
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// healthResponse struct
type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// ServeHTTP method
//
// @Summary Health Check
// @Description Verifies that server is running and responding
// @Tags system
// @Produce json
// @Success 200 {object} healthResponse
// @Router /health [get]
func (handle *HealthHandler) ServeHTTP(writer http.ResponseWriter, _ *http.Request) {
	response := healthResponse{
		Status:  "ok",
		Version: "1.0.0",
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		return
	}
}
