package banner

import (
	"selarashomeid/internal/middleware"

	"github.com/labstack/echo/v4"
)

func (h *handler) Route(v *echo.Group) {
	v.GET("", h.Find)
	v.GET("/:id", h.FindById)
	v.POST("", h.Create, middleware.Authentication)
	v.PUT("/:id", h.Update, middleware.Authentication)
	v.DELETE("/:id", h.Delete, middleware.Authentication)

	// popup
	v.GET("/popup", h.GetPopup)
	v.PUT("/popup/:id", h.UpdatePopup, middleware.Authentication)
	v.PATCH("/popup/:set", h.SetPopup, middleware.Authentication)
}
