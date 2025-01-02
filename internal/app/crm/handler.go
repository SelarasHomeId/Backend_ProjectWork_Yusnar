package crm

import (
	crmaccess "selarashomeid/internal/app/crm/access"
	crmaffiliate "selarashomeid/internal/app/crm/affiliate"
	crmcontact "selarashomeid/internal/app/crm/contact"
	"selarashomeid/internal/factory"
)

type handler struct {
	CrmAccessHandler    crmaccess.Handler
	CrmAffiliateHandler crmaffiliate.Handler
	CrmContactHandler   crmcontact.Handler
}

func NewHandler(f *factory.Factory) *handler {
	return &handler{
		CrmAccessHandler:    *crmaccess.NewHandler(f),
		CrmAffiliateHandler: *crmaffiliate.NewHandler(f),
		CrmContactHandler:   *crmcontact.NewHandler(f),
	}
}
