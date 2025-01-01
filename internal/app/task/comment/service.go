package comment

import (
	"errors"
	"net/http"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/dto"
	"selarashomeid/internal/factory"
	"selarashomeid/internal/model"
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/util/general"
	"selarashomeid/pkg/util/response"
	"selarashomeid/pkg/util/trxmanager"

	"google.golang.org/api/drive/v3"
	"gorm.io/gorm"
)

type Service interface {
	Create(ctx *abstraction.Context, payload *dto.TaskCommentCreateRequest) (map[string]interface{}, error)
	FindByTaskId(ctx *abstraction.Context, payload *dto.TaskCommentFindByTaskIDRequest) (map[string]interface{}, error)
	Delete(ctx *abstraction.Context, payload *dto.TaskCommentDeleteByIDRequest) (map[string]interface{}, error)
	Update(ctx *abstraction.Context, payload *dto.TaskCommentUpdateRequest) (map[string]interface{}, error)
	FindById(ctx *abstraction.Context, payload *dto.TaskCommentFindByIDRequest) (map[string]interface{}, error)
}

type service struct {
	TaskRepository        repository.Task
	TaskCommentRepository repository.TaskComment

	DB     *gorm.DB
	sDrive *drive.Service
	fDrive *drive.File
}

func NewService(f *factory.Factory) Service {
	return &service{
		TaskRepository:        f.TaskRepository,
		TaskCommentRepository: f.TaskCommentRepository,

		DB:     f.Db,
		sDrive: f.GDrive.Service,
		fDrive: f.GDrive.Folder,
	}
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.TaskCommentCreateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		taskData, err := s.TaskRepository.FindById(ctx, *payload.TaskId)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if taskData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "task not found")
		}

		modelTaskComment := &model.TaskCommentEntityModel{
			Context: ctx,
			TaskCommentEntity: model.TaskCommentEntity{
				TaskId:   *payload.TaskId,
				Comment:  *payload.Comment,
				IsDelete: false,
			},
		}

		if err := s.TaskCommentRepository.Create(ctx, modelTaskComment).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		newTaskData := new(model.TaskEntityModel)
		newTaskData.Context = ctx
		newTaskData.ID = *payload.TaskId
		newTaskData.UpdatedAt = general.NowLocal()
		if err = s.TaskRepository.Update(ctx, newTaskData).Error; err != nil {
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

func (s *service) FindByTaskId(ctx *abstraction.Context, payload *dto.TaskCommentFindByTaskIDRequest) (map[string]interface{}, error) {
	var res []map[string]interface{}

	taskData, err := s.TaskRepository.FindById(ctx, payload.TaskId)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	if taskData == nil {
		return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "task not found")
	}

	data, err := s.TaskCommentRepository.FindByTaskId(ctx, payload.TaskId)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	count, err := s.TaskCommentRepository.CountByTaskId(ctx, payload.TaskId)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}

	for _, v := range data {
		res = append(res, map[string]interface{}{
			"id":         v.ID,
			"task_id":    v.TaskId,
			"comment":    v.Comment,
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

func (s *service) Delete(ctx *abstraction.Context, payload *dto.TaskCommentDeleteByIDRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		taskCommentData, err := s.TaskCommentRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if taskCommentData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "task comment not found")
		}

		newTaskCommentData := new(model.TaskCommentEntityModel)
		newTaskCommentData.Context = ctx
		newTaskCommentData.ID = payload.ID
		newTaskCommentData.IsDelete = true
		if err = s.TaskCommentRepository.Update(ctx, newTaskCommentData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		newTaskData := new(model.TaskEntityModel)
		newTaskData.Context = ctx
		newTaskData.ID = taskCommentData.TaskId
		newTaskData.UpdatedAt = general.NowLocal()
		if err = s.TaskRepository.Update(ctx, newTaskData).Error; err != nil {
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

func (s *service) Update(ctx *abstraction.Context, payload *dto.TaskCommentUpdateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		taskCommentData, err := s.TaskCommentRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if taskCommentData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "task comment not found")
		}

		newTaskCommentData := new(model.TaskCommentEntityModel)
		newTaskCommentData.Context = ctx
		newTaskCommentData.ID = payload.ID
		if payload.Comment != nil {
			newTaskCommentData.Comment = *payload.Comment
		}
		if err = s.TaskCommentRepository.Update(ctx, newTaskCommentData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		newTaskData := new(model.TaskEntityModel)
		newTaskData.Context = ctx
		newTaskData.ID = taskCommentData.TaskId
		newTaskData.UpdatedAt = general.NowLocal()
		if err = s.TaskRepository.Update(ctx, newTaskData).Error; err != nil {
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

func (s *service) FindById(ctx *abstraction.Context, payload *dto.TaskCommentFindByIDRequest) (map[string]interface{}, error) {
	var res map[string]interface{} = nil

	data, err := s.TaskCommentRepository.FindById(ctx, payload.ID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	if data != nil {
		res = map[string]interface{}{
			"id":         data.ID,
			"task_id":    data.TaskId,
			"comment":    data.Comment,
			"is_delete":  data.IsDelete,
			"created_at": data.CreatedAt,
			"updated_at": data.UpdatedAt,
		}
	}

	return map[string]interface{}{
		"data": res,
	}, nil
}
