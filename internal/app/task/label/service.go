package label

import (
	"errors"
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
	Create(ctx *abstraction.Context, payload *dto.TaskLabelCreateRequest) (map[string]interface{}, error)
	Find(ctx *abstraction.Context) (map[string]interface{}, error)
	FindById(ctx *abstraction.Context, payload *dto.TaskLabelFindByIDRequest) (map[string]interface{}, error)
	Update(ctx *abstraction.Context, payload *dto.TaskLabelUpdateRequest) (map[string]interface{}, error)
	Delete(ctx *abstraction.Context, payload *dto.TaskLabelDeleteByIDRequest) (map[string]interface{}, error)
}

type service struct {
	TaskLabelRepository repository.TaskLabel
	TaskRepository      repository.Task

	DB *gorm.DB
}

func NewService(f *factory.Factory) Service {
	return &service{
		TaskLabelRepository: f.TaskLabelRepository,
		TaskRepository:      f.TaskRepository,

		DB: f.Db,
	}
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.TaskLabelCreateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		newTaskLabelData := new(model.TaskLabelEntityModel)
		newTaskLabelData.Context = ctx
		newTaskLabelData.Color = payload.Color
		if payload.Title != nil {
			newTaskLabelData.Title = *payload.Title
		}
		if err := s.TaskLabelRepository.Create(ctx, newTaskLabelData).Error; err != nil {
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
	data, err := s.TaskLabelRepository.Find(ctx, false)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	count, err := s.TaskLabelRepository.Count(ctx)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	var res []map[string]interface{} = nil
	for _, v := range data {
		res = append(res, map[string]interface{}{
			"id":         v.ID,
			"title":      v.Title,
			"color":      v.Color,
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

func (s *service) FindById(ctx *abstraction.Context, payload *dto.TaskLabelFindByIDRequest) (map[string]interface{}, error) {
	var res map[string]interface{} = nil

	data, err := s.TaskLabelRepository.FindById(ctx, payload.ID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	if data != nil {
		res = map[string]interface{}{
			"id":         data.ID,
			"title":      data.Title,
			"color":      data.Color,
			"is_delete":  data.IsDelete,
			"created_at": data.CreatedAt,
			"updated_at": data.UpdatedAt,
		}
	}

	return map[string]interface{}{
		"data": res,
	}, nil
}

func (s *service) Update(ctx *abstraction.Context, payload *dto.TaskLabelUpdateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		taskLabelData, err := s.TaskLabelRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if taskLabelData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "task label not found")
		}

		newTaskLabelData := new(model.TaskLabelEntityModel)
		newTaskLabelData.Context = ctx
		newTaskLabelData.ID = payload.ID
		if payload.Title != nil {
			newTaskLabelData.Title = *payload.Title
		}
		if payload.Color != nil {
			newTaskLabelData.Color = *payload.Color
		}
		if err = s.TaskLabelRepository.Update(ctx, newTaskLabelData).Error; err != nil {
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

func (s *service) Delete(ctx *abstraction.Context, payload *dto.TaskLabelDeleteByIDRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		taskLabelData, err := s.TaskLabelRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if taskLabelData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "task label not found")
		}

		newTaskLabelData := new(model.TaskLabelEntityModel)
		newTaskLabelData.Context = ctx
		newTaskLabelData.ID = payload.ID
		newTaskLabelData.IsDelete = true
		if err = s.TaskLabelRepository.Update(ctx, newTaskLabelData).Error; err != nil {
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
