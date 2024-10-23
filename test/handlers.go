package test

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// ServiceHealth defines the health of the service.
type ServiceHealth struct {
	ServiceStatus ServiceHealthStatus      `json:"status"`
	Description   ServiceHealthDescription `json:"description"`
	CompletedAt   time.Time                `json:"completed_at"`
}

// ServiceHealthStatus defines the status of the service.
type ServiceHealthStatus string // @name serviceHealthStatus

const (
	statusHealthy ServiceHealthStatus = "healthy"
)

// ServiceHealthDescription describes the state of the service status.
type ServiceHealthDescription string // @name serviceHealthDescription

const (
	descriptionHealthy ServiceHealthDescription = "everything is awesome"
)

// Handler defines the http router implementation for health endpoints.
type Handler struct{}

// NewHandler creates a new Handler for health endpoints.
func NewHandler() *Handler {
	return &Handler{}
}

// Ready performs readiness check.
//
// @Summary Ready healthcheck
// @Description Performs readiness check
// @Tags health
// @ID health-ready
// @Success 200 "OK"
// @Failure 503 {object} ServiceHealth "Service Unavailable"
// @Router /health/ready [get]
func (h *Handler) Ready(ctx echo.Context) error {
	return ctx.NoContent(http.StatusOK)
}

// Live performs a live check.
//
// @Summary Live healthcheck
// @Description Performs a live check
// @Tags health
// @ID health-live
// @Produce json
// @Success 200 {object} ServiceHealth "OK"
// @Failure 503 {object} ServiceHealth "Service Unavailable"
// @Router /health/live [get]
func (h *Handler) Live(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, ServiceHealth{
		ServiceStatus: statusHealthy,
		Description:   descriptionHealthy,
	})
}
