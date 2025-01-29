package item

import (
	"selarashomeid/internal/middleware"

	"github.com/labstack/echo/v4"
)

func (h *Handler) Route(v *echo.Group) {
	v.POST("", h.Create, middleware.Authentication)
	v.GET("/:task_checklist_id", h.FindByTaskChecklistId, middleware.Authentication)
	v.PUT("/:id", h.Update, middleware.Authentication)
	v.DELETE("/:id", h.Delete, middleware.Authentication)
	v.PATCH("/convert_to_task/:id", h.ConvertToTask, middleware.Authentication)
}
