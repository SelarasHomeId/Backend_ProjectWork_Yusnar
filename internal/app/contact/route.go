package contact

import (
	"selarashomeid/internal/middleware"

	"github.com/labstack/echo/v4"
)

func (h *handler) Route(v *echo.Group) {
	v.GET("/post", h.Create) // for url hit
	v.GET("", h.Find, middleware.Authentication)
	v.GET("/:id", h.FindByID, middleware.Authentication)
	v.DELETE("/:id", h.DeleteByID, middleware.Authentication)
	v.GET("/export", h.Export, middleware.Authentication)
}
