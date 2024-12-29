package board

import (
	"selarashomeid/internal/middleware"

	"github.com/labstack/echo/v4"
)

func (h *handler) Route(v *echo.Group) {
	v.POST("", h.Create, middleware.Authentication)
	v.DELETE("/:id", h.Delete, middleware.Authentication)
	v.PUT("/:id", h.Update, middleware.Authentication)
	v.GET("/:workspace_id", h.FindByWorkspaceId, middleware.Authentication)
}
