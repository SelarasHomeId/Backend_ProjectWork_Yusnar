package comment

import (
	"selarashomeid/internal/middleware"

	"github.com/labstack/echo/v4"
)

func (h *Handler) Route(v *echo.Group) {
	v.POST("", h.Create, middleware.Authentication)
	v.GET("/:task_id", h.FindByTaskId, middleware.Authentication)
	v.DELETE("/:id", h.Delete, middleware.Authentication)
	v.PUT("/:id", h.Update, middleware.Authentication)
}
