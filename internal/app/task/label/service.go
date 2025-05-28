package label

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
	"selarashomeid/pkg/ws"

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
	TaskLabelRepository  repository.TaskLabel
	TaskRepository       repository.Task
	UserRepository       repository.User
	NotifikasiRepository repository.Notifikasi

	DB *gorm.DB
}

func NewService(f *factory.Factory) Service {
	return &service{
		TaskLabelRepository:  f.TaskLabelRepository,
		TaskRepository:       f.TaskRepository,
		UserRepository:       f.UserRepository,
		NotifikasiRepository: f.NotifikasiRepository,

		DB: f.Db,
	}
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.TaskLabelCreateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		userLogin, err := s.UserRepository.FindById(ctx, ctx.Auth.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		userAdmin, err := s.UserRepository.FindByRoleIdArr(ctx, constant.ROLE_ID_ADMIN, true)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		newTaskLabelData := new(model.TaskLabelEntityModel)
		newTaskLabelData.Context = ctx
		newTaskLabelData.Color = payload.Color
		if payload.Title != nil {
			newTaskLabelData.Title = *payload.Title
		}
		if err := s.TaskLabelRepository.Create(ctx, newTaskLabelData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		for _, v := range userAdmin {
			if v.ID != userLogin.ID {
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("%s telah membuat label baru", userLogin.Name)
				modelNotifikasi.Message = fmt.Sprintf("Label: %s, Warna: %s", newTaskLabelData.Title, general.GetColorNameFromCode(newTaskLabelData.Color))
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = v.ID
				modelNotifikasi.TaskId = constant.BLANK_TASK_ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}

				if err := ws.PublishNotificationWithoutTransaction(v.ID, s.DB, ctx); err != nil {
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
			"created_at": general.FormatWithZWithoutChangingTime(v.CreatedAt),
			"updated_at": general.FormatWithZWithoutChangingTime(*v.UpdatedAt),
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
			"created_at": general.FormatWithZWithoutChangingTime(data.CreatedAt),
			"updated_at": general.FormatWithZWithoutChangingTime(*data.UpdatedAt),
		}
	}

	return map[string]interface{}{
		"data": res,
	}, nil
}

func (s *service) Update(ctx *abstraction.Context, payload *dto.TaskLabelUpdateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		userLogin, err := s.UserRepository.FindById(ctx, ctx.Auth.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		userAdmin, err := s.UserRepository.FindByRoleIdArr(ctx, constant.ROLE_ID_ADMIN, true)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		taskLabelData, err := s.TaskLabelRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if taskLabelData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "task label not found")
		}

		var messageNotif []string
		newTaskLabelData := new(model.TaskLabelEntityModel)
		newTaskLabelData.Context = ctx
		newTaskLabelData.ID = payload.ID
		if payload.Title != nil {
			newTaskLabelData.Title = *payload.Title
			if taskLabelData.Title != newTaskLabelData.Title {
				messageNotif = append(messageNotif, fmt.Sprintf("Label berganti nama menjadi %s", newTaskLabelData.Title))
			}
		}
		if payload.Color != nil {
			newTaskLabelData.Color = *payload.Color
			if taskLabelData.Color != newTaskLabelData.Color {
				messageNotif = append(messageNotif, fmt.Sprintf("Label berganti warna menjadi %s", general.GetColorNameFromCode(newTaskLabelData.Color)))
			}
		}
		if err = s.TaskLabelRepository.Update(ctx, newTaskLabelData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		for _, v := range userAdmin {
			if v.ID != userLogin.ID {
				for _, d := range messageNotif {
					modelNotifikasi := new(model.NotifikasiEntityModel)
					modelNotifikasi.Context = ctx
					modelNotifikasi.Title = fmt.Sprintf("%s telah mengupdate label %s", userLogin.Name, taskLabelData.Title)
					modelNotifikasi.Message = d
					modelNotifikasi.IsRead = false
					modelNotifikasi.UserId = v.ID
					modelNotifikasi.TaskId = constant.BLANK_TASK_ID
					if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
						return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
					}

					if err := ws.PublishNotificationWithoutTransaction(v.ID, s.DB, ctx); err != nil {
						return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
					}
				}
			}
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
		userLogin, err := s.UserRepository.FindById(ctx, ctx.Auth.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		userAdmin, err := s.UserRepository.FindByRoleIdArr(ctx, constant.ROLE_ID_ADMIN, true)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

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

		for _, v := range userAdmin {
			if v.ID != userLogin.ID {
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("%s telah menghapus label", userLogin.Name)
				modelNotifikasi.Message = fmt.Sprintf("Label: %s, Warna: %s", taskLabelData.Title, general.GetColorNameFromCode(taskLabelData.Color))
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = v.ID
				modelNotifikasi.TaskId = constant.BLANK_TASK_ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}

				if err := ws.PublishNotificationWithoutTransaction(v.ID, s.DB, ctx); err != nil {
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
