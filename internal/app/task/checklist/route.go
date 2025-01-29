package checklist

import (
	"selarashomeid/internal/middleware"

	"github.com/labstack/echo/v4"
)

func (h *Handler) Route(v *echo.Group) {
	v.POST("", h.Create, middleware.Authentication)
	v.GET("/:task_id", h.FindByTaskId, middleware.Authentication)
	v.PUT("/:id", h.Update, middleware.Authentication)
	v.DELETE("/:id", h.Delete, middleware.Authentication)
	v.GET("", h.Find, middleware.Authentication)

	h.ChecklistItemHandler.Route(v.Group("/item"))
}
