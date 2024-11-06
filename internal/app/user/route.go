package contact

import (
	"github.com/labstack/echo/v4"
)

func (h *handler) Route(v *echo.Group) {
	v.POST("", h.Create)
	v.GET("", h.Find)
	v.GET("/:id", h.FindById)
	v.PUT("/:id", h.Update)
	v.DELETE("/:id", h.Delete)
	v.POST("/change-password/:id", h.ChangePassword)
	v.POST("/reset-password/:id", h.ResetPassword)
}
