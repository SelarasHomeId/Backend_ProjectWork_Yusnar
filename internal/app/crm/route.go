package crm

import "github.com/labstack/echo/v4"

func (h *handler) Route(v *echo.Group) {
	h.CrmAccessHandler.Route(v.Group("/access"))
	h.CrmAffiliateHandler.Route(v.Group("/affiliate"))
	h.CrmContactHandler.Route(v.Group("/contact"))
}
