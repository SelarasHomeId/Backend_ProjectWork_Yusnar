package access

import (
	"net/http"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/dto"
	"selarashomeid/internal/factory"
	"selarashomeid/internal/model"
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/util/response"
	"selarashomeid/pkg/util/trxmanager"

	"gorm.io/gorm"
)

type Service interface {
	Create(ctx *abstraction.Context, payload *dto.AccessCreateRequest) (*dto.AccessCreateResponse, error)
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

func (s *service) Create(ctx *abstraction.Context, payload *dto.AccessCreateRequest) (*dto.AccessCreateResponse, error) {
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
		return nil
	}); err != nil {
		return nil, err
	}
	return &dto.AccessCreateResponse{
		Module:  payload.Module,
		Phone:   payload.Phone,
		Message: payload.Message,
	}, nil
}

func (s *service) Count(ctx *abstraction.Context) (*dto.AccesstCountResponse, error) {
	data, err := s.AccessRepository.Count(ctx)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	return data, nil
}
