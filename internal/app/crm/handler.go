package crm

import (
	"selarashomeid/internal/abstraction"
	crmaccess "selarashomeid/internal/app/crm/access"
	crmaffiliate "selarashomeid/internal/app/crm/affiliate"
	crmcontact "selarashomeid/internal/app/crm/contact"
	"selarashomeid/internal/factory"
	"selarashomeid/pkg/util/response"

	"github.com/labstack/echo/v4"
)

type handler struct {
	service Service

	CrmAccessHandler    crmaccess.Handler
	CrmAffiliateHandler crmaffiliate.Handler
	CrmContactHandler   crmcontact.Handler
}

func NewHandler(f *factory.Factory) *handler {
	return &handler{
		service: NewService(f),

		CrmAccessHandler:    *crmaccess.NewHandler(f),
		CrmAffiliateHandler: *crmaffiliate.NewHandler(f),
		CrmContactHandler:   *crmcontact.NewHandler(f),
	}
}

func (h handler) CalculateTask(c echo.Context) (err error) {
	data, err := h.service.CalculateTask(c.(*abstraction.Context))
	if err != nil {
		return response.ErrorResponse(err).SendError(c)
	}
	return response.SuccessResponse(data).SendSuccess(c)
}
