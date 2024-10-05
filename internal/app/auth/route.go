package auth

import (
	"selarashomeid/internal/middleware"

	"github.com/labstack/echo/v4"
)

func (h *handler) Route(v *echo.Group) {
	v.POST("/login", h.Login)
	v.POST("/logout", h.Logout, middleware.Logout)
	v.POST("/change-password/:id", h.ChangePassword, middleware.Authentication)
	v.POST("/reset-password/:id", h.ResetPassword, middleware.Authentication)
}
