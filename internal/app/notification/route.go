package notification

import (
	"selarashomeid/internal/middleware"

	"github.com/labstack/echo/v4"
)

func (h *handler) Route(v *echo.Group) {
	v.GET("", h.Find, middleware.Authentication)
	v.GET("/count-unread", h.CountUnread, middleware.Authentication)
	v.PATCH("/set-read/:id", h.SetRead, middleware.Authentication)
	v.GET("/:id", h.FindByID, middleware.Authentication)
}
