package access

import (
	"github.com/labstack/echo/v4"
)

func (h *Handler) Route(v *echo.Group) {
	v.GET("/:module", h.Create)
	v.GET("/:module/:phone", h.Create)
	v.GET("/:module/:phone/:message", h.Create)
	v.GET("/count", h.Count)
}
