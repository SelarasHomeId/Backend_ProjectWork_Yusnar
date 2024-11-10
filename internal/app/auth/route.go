package auth

import (
	"selarashomeid/internal/middleware"

	"github.com/labstack/echo/v4"
)

func (h *handler) Route(v *echo.Group) {
	v.POST("/login", h.Login)
	v.POST("/logout", h.Logout, middleware.Logout)
	v.POST("/refresh-token", h.RefreshToken, middleware.RefreshToken)
	v.POST("/send-email/reset-password", h.SendEmailResetPassword)
}
