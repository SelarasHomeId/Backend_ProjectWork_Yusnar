package crm

import (
	"selarashomeid/internal/middleware"

	"github.com/labstack/echo/v4"
)

func (h *handler) Route(v *echo.Group) {
	v.GET("/calculate_task", h.CalculateTask, middleware.Authentication)

	h.CrmAccessHandler.Route(v.Group("/access"))
	h.CrmAffiliateHandler.Route(v.Group("/affiliate"))
	h.CrmContactHandler.Route(v.Group("/contact"))
}
