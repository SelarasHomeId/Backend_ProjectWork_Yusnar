package checklist

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
	"strconv"

	"gorm.io/gorm"
)

type Service interface {
	Create(ctx *abstraction.Context, payload *dto.TaskChecklistCreateRequest) (map[string]interface{}, error)
	FindByTaskId(ctx *abstraction.Context, payload *dto.TaskChecklistFindByTaskIDRequest) (map[string]interface{}, error)
	Update(ctx *abstraction.Context, payload *dto.TaskChecklistUpdateRequest) (map[string]interface{}, error)
	Delete(ctx *abstraction.Context, payload *dto.TaskChecklistDeleteByIDRequest) (map[string]interface{}, error)
	Find(ctx *abstraction.Context) (map[string]interface{}, error)
}

type service struct {
	TaskChecklistRepository repository.TaskChecklist
	TaskRepository          repository.Task
	ChecklistItemRepository repository.ChecklistItem
	UserRepository          repository.User
	TaskCommentRepository   repository.TaskComment
	NotifikasiRepository    repository.Notifikasi

	DB *gorm.DB
}

func NewService(f *factory.Factory) Service {
	return &service{
		TaskChecklistRepository: f.TaskChecklistRepository,
		TaskRepository:          f.TaskRepository,
		ChecklistItemRepository: f.ChecklistItemRepository,
		UserRepository:          f.UserRepository,
		TaskCommentRepository:   f.TaskCommentRepository,
		NotifikasiRepository:    f.NotifikasiRepository,

		DB: f.Db,
	}
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.TaskChecklistCreateRequest) (map[string]interface{}, error) {
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

		modelTaskChecklist := &model.TaskChecklistEntityModel{
			Context: ctx,
			TaskChecklistEntity: model.TaskChecklistEntity{
				TaskId:   payload.TaskId,
				Title:    payload.Title,
				IsDelete: false,
			},
		}
		if err := s.TaskChecklistRepository.Create(ctx, modelTaskChecklist).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		for _, v := range userAdmin {
			if v.ID != ctx.Auth.ID {
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("%s menambahkan checklist pada tugas (%s)", userLogin.Name, taskData.Title)
				modelNotifikasi.Message = modelTaskChecklist.Title
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
			modelNotifikasi.Title = fmt.Sprintf("%s menambahkan checklist pada tugas yang anda buat", userLogin.Name)
			modelNotifikasi.Message = modelTaskChecklist.Title
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
				modelNotifikasi.Title = fmt.Sprintf("%s menambahkan checklist pada tugas (%s)", userLogin.Name, taskData.Title)
				modelNotifikasi.Message = modelTaskChecklist.Title
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = v.ID
				modelNotifikasi.TaskId = taskData.ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
		}

		if err := s.TaskCommentRepository.CreateHistory(ctx, taskData.ID, fmt.Sprintf("Checklist (%s) telah ditambahkan oleh %s", payload.Title, userLogin.Name)).Error; err != nil {
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

func (s *service) FindByTaskId(ctx *abstraction.Context, payload *dto.TaskChecklistFindByTaskIDRequest) (map[string]interface{}, error) {
	var res []map[string]interface{} = nil

	taskData, err := s.TaskRepository.FindById(ctx, payload.TaskId)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	if taskData == nil {
		return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "task not found")
	}

	data, err := s.TaskChecklistRepository.FindByTaskId(ctx, payload.TaskId, false)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	count, err := s.TaskChecklistRepository.CountByTaskId(ctx, payload.TaskId)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}

	for _, v := range data {
		dataChecklistItem, err := s.ChecklistItemRepository.FindByTaskChecklistIdArr(ctx, v.ID, true)
		if err != nil && err.Error() != "record not found" {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		var dataChecklistItemArr []map[string]interface{}
		isChecklistItemCompleted := 0
		for _, ci := range dataChecklistItem {
			if ci.IsCompleted {
				isChecklistItemCompleted++
			}
			checklistItem := map[string]interface{}{
				"id":             ci.ID,
				"title":          ci.Title,
				"assign_to_user": ci.AssignToUser,
				"due_date":       ci.DueDate,
				"is_completed":   ci.IsCompleted,
				"sort_number":    ci.SortNumber,
				"is_delete":      ci.IsDelete,
				"created_at":     general.FormatWithZWithoutChangingTime(ci.CreatedAt),
				"updated_at":     general.FormatWithZWithoutChangingTime(*ci.UpdatedAt),
			}

			if ci.AssignToUser != nil {
				var assignToUser []map[string]interface{}
				assignToUserArr := general.StringToArrayInt(ci.AssignToUser)
				for _, v := range assignToUserArr {
					dataUser, err := s.UserRepository.FindById(ctx, v)
					if err != nil && err.Error() != "record not found" {
						return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
					}
					if dataUser != nil {
						assignToUser = append(assignToUser, map[string]interface{}{
							"id":     dataUser.ID,
							"name":   dataUser.Name,
							"email":  dataUser.Email,
							"divisi": dataUser.Divisi.Name,
							"role":   dataUser.Role.Name,
						})
					}
				}
				checklistItem["assign_to_user"] = map[string]interface{}{
					"count": len(assignToUserArr),
					"data":  assignToUser,
				}
			}

			dataChecklistItemArr = append(dataChecklistItemArr, checklistItem)
		}

		checkPersentase := 0
		if len(dataChecklistItem) > 0 {
			checkPersentase = (100 * isChecklistItemCompleted) / len(dataChecklistItem)
		}

		res = append(res, map[string]interface{}{
			"id":               v.ID,
			"task_id":          v.TaskId,
			"title":            v.Title,
			"check_persentase": strconv.Itoa(checkPersentase) + "%",
			"is_delete":        v.IsDelete,
			"created_at":       general.FormatWithZWithoutChangingTime(v.CreatedAt),
			"updated_at":       general.FormatWithZWithoutChangingTime(*v.UpdatedAt),
			"item": map[string]interface{}{
				"count": len(dataChecklistItem),
				"data":  dataChecklistItemArr,
			},
		})
	}

	return map[string]interface{}{
		"count": count,
		"data":  res,
	}, nil
}

func (s *service) Update(ctx *abstraction.Context, payload *dto.TaskChecklistUpdateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		taskChecklistData, err := s.TaskChecklistRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if taskChecklistData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "task checklist not found")
		}

		taskData, err := s.TaskRepository.FindById(ctx, taskChecklistData.TaskId)
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

		var messageNotif []string
		newTaskChecklistData := new(model.TaskChecklistEntityModel)
		newTaskChecklistData.Context = ctx
		newTaskChecklistData.ID = payload.ID
		if payload.Title != nil {
			newTaskChecklistData.Title = *payload.Title
			if taskChecklistData.Title != newTaskChecklistData.Title {
				messageNotif = append(messageNotif, fmt.Sprintf("Checklist berganti nama menjadi (%s)", newTaskChecklistData.Title))
			}
		}
		if err = s.TaskChecklistRepository.Update(ctx, newTaskChecklistData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		newTaskData := new(model.TaskEntityModel)
		newTaskData.Context = ctx
		newTaskData.ID = taskChecklistData.TaskId
		newTaskData.UpdatedAt = general.NowLocal()
		if err = s.TaskRepository.Update(ctx, newTaskData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		for _, v := range userAdmin {
			if v.ID != ctx.Auth.ID {
				for _, d := range messageNotif {
					modelNotifikasi := new(model.NotifikasiEntityModel)
					modelNotifikasi.Context = ctx
					modelNotifikasi.Title = fmt.Sprintf("%s telah mengupdate checklist (%s) di %s", userLogin.Name, taskChecklistData.Title, taskData.Title)
					modelNotifikasi.Message = d
					modelNotifikasi.IsRead = false
					modelNotifikasi.UserId = v.ID
					modelNotifikasi.TaskId = taskData.ID
					if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
						return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
					}
				}
			}
		}

		if taskData.CreateBy.RoleId != constant.ROLE_ID_ADMIN {
			for _, v := range messageNotif {
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("Tugas yang anda buat (%s) terdapat checklist (%s) yang telah diupdate oleh %s", taskData.Title, taskChecklistData.Title, userLogin.Name)
				modelNotifikasi.Message = v
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = taskData.CreateBy.ID
				modelNotifikasi.TaskId = taskData.ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
		}

		for _, v := range assignedMember {
			if v.Role.ID != constant.ROLE_ID_ADMIN && v.ID != taskData.CreateBy.ID {
				for _, d := range messageNotif {
					modelNotifikasi := new(model.NotifikasiEntityModel)
					modelNotifikasi.Context = ctx
					modelNotifikasi.Title = fmt.Sprintf("%s telah mengupdate checklist (%s) di %s", userLogin.Name, taskChecklistData.Title, taskData.Title)
					modelNotifikasi.Message = d
					modelNotifikasi.IsRead = false
					modelNotifikasi.UserId = v.ID
					modelNotifikasi.TaskId = taskData.ID
					if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
						return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
					}
				}
			}
		}

		for _, v := range messageNotif {
			if err := s.TaskCommentRepository.CreateHistory(ctx, taskData.ID, v).Error; err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
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

func (s *service) Delete(ctx *abstraction.Context, payload *dto.TaskChecklistDeleteByIDRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		taskChecklistData, err := s.TaskChecklistRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if taskChecklistData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "task checklist not found")
		}

		taskData, err := s.TaskRepository.FindById(ctx, taskChecklistData.TaskId)
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

		newTaskChecklistData := new(model.TaskChecklistEntityModel)
		newTaskChecklistData.Context = ctx
		newTaskChecklistData.ID = payload.ID
		newTaskChecklistData.IsDelete = true
		if err = s.TaskChecklistRepository.Update(ctx, newTaskChecklistData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		itemInChecklist, err := s.ChecklistItemRepository.FindByTaskChecklistIdArr(ctx, taskChecklistData.ID, true)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		for _, v := range itemInChecklist {
			newChecklistItemData := new(model.ChecklistItemEntityModel)
			newChecklistItemData.Context = ctx
			newChecklistItemData.ID = v.ID
			newChecklistItemData.IsDelete = true
			if err = s.ChecklistItemRepository.Update(ctx, newChecklistItemData).Error; err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
		}

		newTaskData := new(model.TaskEntityModel)
		newTaskData.Context = ctx
		newTaskData.ID = taskChecklistData.TaskId
		newTaskData.UpdatedAt = general.NowLocal()
		if err = s.TaskRepository.Update(ctx, newTaskData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		for _, v := range userAdmin {
			if v.ID != ctx.Auth.ID {
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("%s menghapus checklist pada tugas (%s)", userLogin.Name, taskData.Title)
				modelNotifikasi.Message = fmt.Sprintf("Checklist yang dihapus: %s", taskChecklistData.Title)
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
			modelNotifikasi.Title = fmt.Sprintf("%s menghaous checklist pada tugas yang anda buat", userLogin.Name)
			modelNotifikasi.Message = fmt.Sprintf("Checklist yang dihapus: %s", taskChecklistData.Title)
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
				modelNotifikasi.Title = fmt.Sprintf("%s menghapus checklist pada tugas (%s)", userLogin.Name, taskData.Title)
				modelNotifikasi.Message = fmt.Sprintf("Checklist yang dihapus: %s", taskChecklistData.Title)
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = v.ID
				modelNotifikasi.TaskId = taskData.ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
		}

		if err := s.TaskCommentRepository.CreateHistory(ctx, taskChecklistData.TaskId, fmt.Sprintf("Checklist: (%s) dihapus oleh %s", taskChecklistData.Title, userLogin.Name)).Error; err != nil {
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

func (s *service) Find(ctx *abstraction.Context) (map[string]interface{}, error) {
	var res []map[string]interface{} = nil

	data, err := s.TaskChecklistRepository.Find(ctx, false)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	count, err := s.TaskChecklistRepository.Count(ctx)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}

	for _, v := range data {
		taskData, err := s.TaskRepository.FindById(ctx, v.TaskId)
		if err != nil && err.Error() != "record not found" {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		dataChecklistItem, err := s.ChecklistItemRepository.FindByTaskChecklistIdArr(ctx, v.ID, true)
		if err != nil && err.Error() != "record not found" {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		var dataChecklistItemArr []map[string]interface{}
		isChecklistItemCompleted := 0
		for _, ci := range dataChecklistItem {
			if ci.IsCompleted {
				isChecklistItemCompleted++
			}
			checklistItem := map[string]interface{}{
				"id":             ci.ID,
				"title":          ci.Title,
				"assign_to_user": ci.AssignToUser,
				"due_date":       ci.DueDate,
				"is_completed":   ci.IsCompleted,
				"sort_number":    ci.SortNumber,
				"is_delete":      ci.IsDelete,
				"created_at":     general.FormatWithZWithoutChangingTime(ci.CreatedAt),
				"updated_at":     general.FormatWithZWithoutChangingTime(*ci.UpdatedAt),
			}

			if ci.AssignToUser != nil {
				var assignToUser []map[string]interface{}
				assignToUserArr := general.StringToArrayInt(ci.AssignToUser)
				for _, v := range assignToUserArr {
					dataUser, err := s.UserRepository.FindById(ctx, v)
					if err != nil && err.Error() != "record not found" {
						return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
					}
					if dataUser != nil {
						assignToUser = append(assignToUser, map[string]interface{}{
							"id":     dataUser.ID,
							"name":   dataUser.Name,
							"email":  dataUser.Email,
							"divisi": dataUser.Divisi.Name,
							"role":   dataUser.Role.Name,
						})
					}
				}
				checklistItem["assign_to_user"] = map[string]interface{}{
					"count": len(assignToUserArr),
					"data":  assignToUser,
				}
			}

			dataChecklistItemArr = append(dataChecklistItemArr, checklistItem)
		}

		checkPersentase := 0
		if len(dataChecklistItem) > 0 {
			checkPersentase = (100 * isChecklistItemCompleted) / len(dataChecklistItem)
		}

		res = append(res, map[string]interface{}{
			"id": v.ID,
			"task": map[string]interface{}{
				"id":    taskData.ID,
				"title": taskData.Title,
			},
			"title":            v.Title,
			"check_persentase": strconv.Itoa(checkPersentase) + "%",
			"is_delete":        v.IsDelete,
			"created_at":       general.FormatWithZWithoutChangingTime(v.CreatedAt),
			"updated_at":       general.FormatWithZWithoutChangingTime(*v.UpdatedAt),
			"item": map[string]interface{}{
				"count": len(dataChecklistItem),
				"data":  dataChecklistItemArr,
			},
		})
	}

	return map[string]interface{}{
		"count": count,
		"data":  res,
	}, nil
}
