package comment

import (
	"errors"
	"fmt"
	"net/http"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/dto"
	"selarashomeid/internal/factory"
	"selarashomeid/internal/model"
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/constant"
	"selarashomeid/pkg/util/general"
	"selarashomeid/pkg/util/response"
	"selarashomeid/pkg/util/trxmanager"

	"gorm.io/gorm"
)

type Service interface {
	Create(ctx *abstraction.Context, payload *dto.TaskCommentCreateRequest) (map[string]interface{}, error)
	FindByTaskId(ctx *abstraction.Context, payload *dto.TaskCommentFindByTaskIDRequest) (map[string]interface{}, error)
	Delete(ctx *abstraction.Context, payload *dto.TaskCommentDeleteByIDRequest) (map[string]interface{}, error)
	Update(ctx *abstraction.Context, payload *dto.TaskCommentUpdateRequest) (map[string]interface{}, error)
}

type service struct {
	TaskRepository        repository.Task
	TaskCommentRepository repository.TaskComment
	UserRepository        repository.User
	NotifikasiRepository  repository.Notifikasi

	DB *gorm.DB
}

func NewService(f *factory.Factory) Service {
	return &service{
		TaskRepository:        f.TaskRepository,
		TaskCommentRepository: f.TaskCommentRepository,
		UserRepository:        f.UserRepository,
		NotifikasiRepository:  f.NotifikasiRepository,

		DB: f.Db,
	}
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.TaskCommentCreateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		taskData, err := s.TaskRepository.FindById(ctx, payload.TaskId)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if taskData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "task not found")
		}

		userLogin, err := s.UserRepository.FindById(ctx, ctx.Auth.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		userAdmin, err := s.UserRepository.FindByRoleIdArr(ctx, constant.ROLE_ID_ADMIN, true)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		var assignedMember []*model.UserEntityModel
		if taskData.AssignToUser != nil {
			assignToUserArr := general.StringToArrayInt(taskData.AssignToUser)
			for _, v := range assignToUserArr {
				dataUser, err := s.UserRepository.FindById(ctx, v)
				if err != nil && err.Error() != "record not found" {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
				if dataUser != nil {
					assignedMember = append(assignedMember, dataUser)
				}
			}
		}

		modelTaskComment := &model.TaskCommentEntityModel{
			Context: ctx,
			TaskCommentEntity: model.TaskCommentEntity{
				TaskId:    payload.TaskId,
				Comment:   payload.Comment,
				IsDelete:  false,
				IsHistory: false,
			},
		}

		if err := s.TaskCommentRepository.Create(ctx, modelTaskComment).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		newTaskData := new(model.TaskEntityModel)
		newTaskData.Context = ctx
		newTaskData.ID = payload.TaskId
		newTaskData.UpdatedAt = general.NowLocal()
		if err = s.TaskRepository.Update(ctx, newTaskData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		for _, v := range userAdmin {
			if v.ID != ctx.Auth.ID {
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("%s menambahkan komentar pada tugas (%s)", userLogin.Name, taskData.Title)
				modelNotifikasi.Message = modelTaskComment.Comment
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = v.ID
				modelNotifikasi.TaskId = taskData.ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
		}

		if taskData.CreateBy.RoleId != constant.ROLE_ID_ADMIN {
			modelNotifikasi := new(model.NotifikasiEntityModel)
			modelNotifikasi.Context = ctx
			modelNotifikasi.Title = fmt.Sprintf("%s menambahkan komentar pada tugas yang anda buat", userLogin.Name)
			modelNotifikasi.Message = modelTaskComment.Comment
			modelNotifikasi.IsRead = false
			modelNotifikasi.UserId = taskData.CreateBy.ID
			modelNotifikasi.TaskId = taskData.ID
			if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
		}

		for _, v := range assignedMember {
			if v.Role.ID != constant.ROLE_ID_ADMIN && v.ID != taskData.CreateBy.ID {
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("%s menambahkan komentar pada tugas (%s)", userLogin.Name, taskData.Title)
				modelNotifikasi.Message = modelTaskComment.Comment
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = v.ID
				modelNotifikasi.TaskId = taskData.ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
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
	var res []map[string]interface{} = nil

	taskData, err := s.TaskRepository.FindById(ctx, payload.TaskId)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	if taskData == nil {
		return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "task not found")
	}

	data, err := s.TaskCommentRepository.FindByTaskId(ctx, payload.TaskId, false)
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
			"is_history": v.IsHistory,
			"created_at": general.FormatWithZWithoutChangingTime(v.CreatedAt),
			"updated_at": general.FormatWithZWithoutChangingTime(*v.UpdatedAt),
			"created_by": map[string]interface{}{
				"id":    v.CreateBy.ID,
				"name":  v.CreateBy.Name,
				"email": v.CreateBy.Email,
			},
			"updated_by": map[string]interface{}{
				"id":    v.UpdateBy.ID,
				"name":  v.UpdateBy.Name,
				"email": v.UpdateBy.Email,
			},
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

		taskData, err := s.TaskRepository.FindById(ctx, taskCommentData.TaskId)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		userLogin, err := s.UserRepository.FindById(ctx, ctx.Auth.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		userAdmin, err := s.UserRepository.FindByRoleIdArr(ctx, constant.ROLE_ID_ADMIN, true)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		var assignedMember []*model.UserEntityModel
		if taskData.AssignToUser != nil {
			assignToUserArr := general.StringToArrayInt(taskData.AssignToUser)
			for _, v := range assignToUserArr {
				dataUser, err := s.UserRepository.FindById(ctx, v)
				if err != nil && err.Error() != "record not found" {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
				if dataUser != nil {
					assignedMember = append(assignedMember, dataUser)
				}
			}
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

		for _, v := range userAdmin {
			if v.ID != ctx.Auth.ID {
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("%s menghapus komentar pada tugas (%s)", userLogin.Name, taskData.Title)
				modelNotifikasi.Message = fmt.Sprintf("Komentar yang dihapus: %s", taskCommentData.Comment)
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = v.ID
				modelNotifikasi.TaskId = taskData.ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
		}

		if taskData.CreateBy.RoleId != constant.ROLE_ID_ADMIN {
			modelNotifikasi := new(model.NotifikasiEntityModel)
			modelNotifikasi.Context = ctx
			modelNotifikasi.Title = fmt.Sprintf("%s menghapus komentar pada tugas yang anda buat", userLogin.Name)
			modelNotifikasi.Message = fmt.Sprintf("Komentar yang dihapus: %s", taskCommentData.Comment)
			modelNotifikasi.IsRead = false
			modelNotifikasi.UserId = taskData.CreateBy.ID
			modelNotifikasi.TaskId = taskData.ID
			if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
		}

		for _, v := range assignedMember {
			if v.Role.ID != constant.ROLE_ID_ADMIN && v.ID != taskData.CreateBy.ID {
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("%s menghapus komentar pada tugas (%s)", userLogin.Name, taskData.Title)
				modelNotifikasi.Message = fmt.Sprintf("Komentar yang dihapus: %s", taskCommentData.Comment)
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = v.ID
				modelNotifikasi.TaskId = taskData.ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
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
