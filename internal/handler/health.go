package handler

import (
	"net/http"

	"pack_mate/internal/dto/response"
	"pack_mate/internal/service"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles process and dependency health requests.
type HealthHandler struct {
	svc service.HealthService
}

// NewHealthHandler creates a health handler.
func NewHealthHandler(svc service.HealthService) *HealthHandler {
	return &HealthHandler{svc: svc}
}

// Health returns whether the application process is alive.
func (h *HealthHandler) Health(c *gin.Context) {
	if err := h.svc.Health(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, response.HealthResponse{Status: "unavailable"})
		return
	}
	c.JSON(http.StatusOK, response.HealthResponse{Status: "ok"})
}

// Ready returns whether required dependencies are available.
func (h *HealthHandler) Ready(c *gin.Context) {
	if err := h.svc.Ready(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, response.ReadyResponse{
			Status: "not_ready",
			Checks: map[string]string{"mongodb": "unavailable"},
		})
		return
	}
	c.JSON(http.StatusOK, response.ReadyResponse{
		Status: "ready",
		Checks: map[string]string{"mongodb": "ok"},
	})
}
