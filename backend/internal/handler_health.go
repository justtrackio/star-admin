package internal

import (
	"context"

	"github.com/gosoline-project/httpserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

// HandlerHealth reports application availability without requiring a cluster connection.
type HandlerHealth struct{}

// NewHandlerHealth constructs the application health handler.
func NewHandlerHealth(_ context.Context, _ cfg.Config, _ log.Logger) (*HandlerHealth, error) {
	return &HandlerHealth{}, nil
}

// HealthResponse describes the application status, not the StarRocks cluster status.
type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

// GetHealth returns the application status.
func (h *HandlerHealth) GetHealth(_ context.Context) (httpserver.Response, error) {
	return httpserver.NewJsonResponse(HealthResponse{Status: "ok", Service: "star-admin"}), nil
}
