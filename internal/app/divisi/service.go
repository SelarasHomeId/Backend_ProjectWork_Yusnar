package divisi

import (
	"errors"
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
	Create(ctx *abstraction.Context, payload *dto.DivisiCreateRequest) (map[string]interface{}, error)
	Find(ctx *abstraction.Context) (map[string]interface{}, error)
	Update(ctx *abstraction.Context, payload *dto.DivisiUpdateRequest) (map[string]interface{}, error)
	Delete(ctx *abstraction.Context, payload *dto.DivisiDeleteByIDRequest) (map[string]interface{}, error)
}

type service struct {
	DivisiRepository repository.Divisi
	UserRepository   repository.User

	DB *gorm.DB
}

func NewService(f *factory.Factory) Service {
	return &service{
		DivisiRepository: f.DivisiRepository,
		UserRepository:   f.UserRepository,

		DB: f.Db,
	}
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.DivisiCreateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		if ctx.Auth.RoleID != constant.ROLE_ID_ADMIN {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "this role is not permitted")
		}

		dataAllDivisi, err := s.DivisiRepository.Find(ctx)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		for _, v := range dataAllDivisi {
			if *payload.Name == v.Name {
				return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "divisi already exist")
			}
		}

		modelDivisi := &model.DivisiEntityModel{
			Context: ctx,
			DivisiEntity: model.DivisiEntity{
				Name:     *payload.Name,
				IsDelete: false,
			},
		}
		if err := s.DivisiRepository.Create(ctx, modelDivisi).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		return nil
	}); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message": "success create!",
	}, nil
}

func (s *service) Find(ctx *abstraction.Context) (map[string]interface{}, error) {
	var res []map[string]interface{}
	if ctx.Auth.RoleID != constant.ROLE_ID_ADMIN {
		return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "this role is not permitted")
	}
	data, err := s.DivisiRepository.Find(ctx)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	count, err := s.DivisiRepository.Count(ctx)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	for _, v := range data {
		res = append(res, map[string]interface{}{
			"id":         v.ID,
			"name":       v.Name,
			"is_delete":  v.IsDelete,
			"created_at": v.CreatedAt,
			"updated_at": v.UpdatedAt,
		})
	}
	return map[string]interface{}{
		"count": count,
		"data":  res,
	}, nil
}

func (s *service) Update(ctx *abstraction.Context, payload *dto.DivisiUpdateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		if ctx.Auth.RoleID != constant.ROLE_ID_ADMIN {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "this role is not permitted")
		}

		divisiData, err := s.DivisiRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if divisiData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "divisi not found")
		}

		newDivisiData := new(model.DivisiEntityModel)
		newDivisiData.Context = ctx
		newDivisiData.ID = payload.ID
		if payload.Name != nil {
			newDivisiData.Name = *payload.Name
		}

		if err = s.DivisiRepository.Update(ctx, newDivisiData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message": "success update!",
	}, nil
}

func (s *service) Delete(ctx *abstraction.Context, payload *dto.DivisiDeleteByIDRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		if ctx.Auth.RoleID != constant.ROLE_ID_ADMIN {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "this role is not permitted")
		}

		divisiData, err := s.DivisiRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if divisiData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "divisi not found")
		}

		divisiInUserData, err := s.UserRepository.FindByDivisiId(ctx, &payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if divisiInUserData != nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "divisi data is being used")
		}

		newDivisiData := new(model.DivisiEntityModel)
		newDivisiData.Context = ctx
		newDivisiData.ID = divisiData.ID
		newDivisiData.IsDelete = true

		if err = s.DivisiRepository.Update(ctx, newDivisiData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message": "success delete!",
	}, nil
}
