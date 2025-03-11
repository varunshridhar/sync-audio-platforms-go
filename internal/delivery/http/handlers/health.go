package handlers

import (
	"net/http"
	"sync-audio-platforms-go/internal/domain"

	"github.com/labstack/echo/v4"
)

type HealthHandler struct {
	healthUseCase interface {
		Check() domain.HealthStatus
	}
}

func NewHealthHandler(healthUseCase interface {
	Check() domain.HealthStatus
}) *HealthHandler {
	return &HealthHandler{
		healthUseCase: healthUseCase,
	}
}

func (h *HealthHandler) Check(c echo.Context) error {
	status := h.healthUseCase.Check()
	return c.JSON(http.StatusOK, status)
}
