package access

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/dto"
	"selarashomeid/internal/factory"
	"selarashomeid/internal/model"
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/util/general"
	"selarashomeid/pkg/util/response"
	"selarashomeid/pkg/util/trxmanager"

	"gorm.io/gorm"
)

type Service interface {
	Create(ctx *abstraction.Context, payload *dto.AccessCreateRequest) (*dto.AccessCreateRequest, error)
	GetCount(ctx *abstraction.Context) (map[string]interface{}, error)
}

type service struct {
	AccessRepository    repository.Access
	AffiliateRepository repository.Affiliate
	ContactRepository   repository.Contact

	DB *gorm.DB
}

func NewService(f *factory.Factory) Service {
	return &service{
		AccessRepository:    f.AccessRepository,
		AffiliateRepository: f.AffiliateRepository,
		ContactRepository:   f.ContactRepository,

		DB: f.Db,
	}
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.AccessCreateRequest) (data *dto.AccessCreateRequest, err error) {
	if err = trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		modelAccess := &model.AccessEntityModel{
			Context: ctx,
			AccessEntity: model.AccessEntity{
				Module:    *payload.Module,
				CreatedAt: *general.NowLocal(),
			},
		}
		if err = s.AccessRepository.Create(ctx, modelAccess).Error; err != nil {
			return response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return &dto.AccessCreateRequest{
		Module: payload.Module,
		Option: payload.Option,
	}, nil
}

func (s *service) GetCount(ctx *abstraction.Context) (map[string]interface{}, error) {
	var (
		dataSosmed    dto.AccessGetCountResponse
		dataAffiliate dto.AffiliateGetCountResponse
		dataContact   dto.ContactGetCountResponse
		err           error
	)
	if err = trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		dataSosmed, err = s.AccessRepository.GetCount(ctx)
		if err != nil {
			return response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
		}
		dataAffiliate, err = s.AffiliateRepository.GetCountAffiliate(ctx)
		if err != nil {
			return response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
		}
		dataContact, err = s.ContactRepository.GetCountContact(ctx)
		if err != nil {
			return response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"count_instagram": dataSosmed.CountInstagram,
		"count_tiktok":    dataSosmed.CountTiktok,
		"count_facebook":  dataSosmed.CountFacebook,
		"count_whatsapp":  dataSosmed.CountWhatsapp,
		"count_affiliate": dataAffiliate.CountAffiliate,
		"count_contact":   dataContact.CountContact,
	}, nil
}
