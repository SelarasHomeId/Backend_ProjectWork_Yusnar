package divisi

import (
	"selarashomeid/internal/middleware"

	"github.com/labstack/echo/v4"
)

func (h *handler) Route(v *echo.Group) {
	v.POST("", h.Create, middleware.Authentication)
	v.GET("", h.Find, middleware.Authentication)
	v.PUT("/:id", h.Update, middleware.Authentication)
	v.DELETE("/:id", h.Delete, middleware.Authentication)
}
