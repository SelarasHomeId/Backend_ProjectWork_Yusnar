package contact

import (
	"selarashomeid/internal/middleware"

	"github.com/labstack/echo/v4"
)

func (h *Handler) Route(v *echo.Group) {
	v.POST("", h.Create)
	v.GET("", h.Find, middleware.Authentication)
	v.GET("/export", h.Export, middleware.Authentication)
}
