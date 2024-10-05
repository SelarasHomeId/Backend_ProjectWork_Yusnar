package access

import (
	"selarashomeid/internal/middleware"

	"github.com/labstack/echo/v4"
)

func (h *handler) Route(v *echo.Group) {
	v.GET("", h.Create) // for url hit
	v.GET("/count", h.GetCount, middleware.Authentication)
}
