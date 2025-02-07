package access

import (
	"net/http"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/dto"
	"selarashomeid/internal/factory"
	"selarashomeid/internal/model"
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/constant"
	"selarashomeid/pkg/util/response"
	"selarashomeid/pkg/util/trxmanager"

	"gorm.io/gorm"
)

type Service interface {
	Create(ctx *abstraction.Context, payload *dto.AccessCreateRequest) (*string, error)
	Count(ctx *abstraction.Context) (*dto.AccesstCountResponse, error)
}

type service struct {
	AccessRepository repository.Access

	DB *gorm.DB
}

func NewService(f *factory.Factory) Service {
	return &service{
		AccessRepository: f.AccessRepository,

		DB: f.Db,
	}
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.AccessCreateRequest) (*string, error) {
	url := ""
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		modelAccess := &model.AccessEntityModel{
			Context: ctx,
			AccessEntity: model.AccessEntity{
				Module: *payload.Module,
			},
		}
		if err := s.AccessRepository.Create(ctx, modelAccess).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		switch *payload.Module {
		case "instagram":
			url = constant.LINK_INSTAGRAM
		case "tiktok":
			url = constant.LINK_TIKTOK
		case "facebook":
			url = constant.LINK_FACEBOOK
		case "whatsapp":
			url = constant.LINK_WHATSAPP
			if payload.Phone != nil {
				url += *payload.Phone
				if payload.Message != nil {
					url += "?text=" + *payload.Message
				}
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return &url, nil
}

func (s *service) Count(ctx *abstraction.Context) (*dto.AccesstCountResponse, error) {
	data, err := s.AccessRepository.Count(ctx)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	return data, nil
}
