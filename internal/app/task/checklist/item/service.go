package item

import (
	"context"
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

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type Service interface {
	Create(ctx *abstraction.Context, payload *dto.ChecklistItemCreateRequest) (map[string]interface{}, error)
	FindByTaskChecklistId(ctx *abstraction.Context, payload *dto.ChecklistItemFindByChecklistIDRequest) (map[string]interface{}, error)
	Update(ctx *abstraction.Context, payload *dto.ChecklistItemUpdateRequest) (map[string]interface{}, error)
	Delete(ctx *abstraction.Context, payload *dto.ChecklistItemDeleteByIDRequest) (map[string]interface{}, error)
	ConvertToTask(ctx *abstraction.Context, payload *dto.ChecklistItemConvertToTaskRequest) (map[string]interface{}, error)
}

type service struct {
	ChecklistItemRepository repository.ChecklistItem
	TaskChecklistRepository repository.TaskChecklist
	UserRepository          repository.User
	TaskRepository          repository.Task
	BoardRepository         repository.Board
	NotifikasiRepository    repository.Notifikasi
	TaskCommentRepository   repository.TaskComment

	DB      *gorm.DB
	DbRedis *redis.Client
}

func NewService(f *factory.Factory) Service {
	return &service{
		ChecklistItemRepository: f.ChecklistItemRepository,
		TaskChecklistRepository: f.TaskChecklistRepository,
		UserRepository:          f.UserRepository,
		TaskRepository:          f.TaskRepository,
		BoardRepository:         f.BoardRepository,
		NotifikasiRepository:    f.NotifikasiRepository,
		TaskCommentRepository:   f.TaskCommentRepository,

		DB:      f.Db,
		DbRedis: f.DbRedis,
	}
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.ChecklistItemCreateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		taskChecklistData, err := s.TaskChecklistRepository.FindById(ctx, payload.TaskChecklistId)
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

		dataItemInChecklist, err := s.ChecklistItemRepository.FindByTaskChecklistId(ctx, payload.TaskChecklistId)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		sortNum := 1
		if dataItemInChecklist != nil {
			sortNum = dataItemInChecklist.SortNumber + 1
		}

		modelChecklistItem := &model.ChecklistItemEntityModel{
			Context: ctx,
			ChecklistItemEntity: model.ChecklistItemEntity{
				TaskChecklistId: payload.TaskChecklistId,
				Title:           payload.Title,
				IsCompleted:     false,
				IsDelete:        false,
				SortNumber:      sortNum,
			},
		}
		if err := s.ChecklistItemRepository.Create(ctx, modelChecklistItem).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		for _, v := range userAdmin {
			if v.ID != ctx.Auth.ID {
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("%s menambahkan item pada checklist (%s) di tugas (%s)", userLogin.Name, taskChecklistData.Title, taskData.Title)
				modelNotifikasi.Message = modelChecklistItem.Title
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
			modelNotifikasi.Title = fmt.Sprintf("%s menambahkan item pada checklist (%s) di tugas yang anda buat", userLogin.Name, taskChecklistData.Title)
			modelNotifikasi.Message = modelChecklistItem.Title
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
				modelNotifikasi.Title = fmt.Sprintf("%s menambahkan item pada checklist (%s) di tugas (%s)", userLogin.Name, taskChecklistData.Title, taskData.Title)
				modelNotifikasi.Message = modelChecklistItem.Title
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = v.ID
				modelNotifikasi.TaskId = taskData.ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
		}

		if err := s.TaskCommentRepository.CreateHistory(ctx, taskChecklistData.TaskId, fmt.Sprintf("Item (%s) untuk Checklist (%s) telah ditambahkan oleh %s", payload.Title, taskChecklistData.Title, userLogin.Name)).Error; err != nil {
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

func (s *service) FindByTaskChecklistId(ctx *abstraction.Context, payload *dto.ChecklistItemFindByChecklistIDRequest) (map[string]interface{}, error) {
	var res []map[string]interface{} = nil

	checklistData, err := s.TaskChecklistRepository.FindById(ctx, payload.TaskChecklistId)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	if checklistData == nil {
		return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "checklist not found")
	}

	data, err := s.ChecklistItemRepository.FindByTaskChecklistIdArr(ctx, payload.TaskChecklistId, false)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	count, err := s.ChecklistItemRepository.CountByTaskChecklistIdArr(ctx, payload.TaskChecklistId)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}

	for _, v := range data {
		checklistItem := map[string]interface{}{
			"id":             v.ID,
			"title":          v.Title,
			"assign_to_user": v.AssignToUser,
			"due_date":       v.DueDate,
			"is_completed":   v.IsCompleted,
			"sort_number":    v.SortNumber,
			"is_delete":      v.IsDelete,
			"created_at":     general.FormatWithZWithoutChangingTime(v.CreatedAt),
			"updated_at":     general.FormatWithZWithoutChangingTime(*v.UpdatedAt),
		}

		if v.AssignToUser != nil {
			var assignToUser []map[string]interface{}
			assignToUserArr := general.StringToArrayInt(v.AssignToUser)
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

		res = append(res, checklistItem)
	}

	return map[string]interface{}{
		"count": count,
		"data":  res,
	}, nil
}

func (s *service) Update(ctx *abstraction.Context, payload *dto.ChecklistItemUpdateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		checklistItemData, err := s.ChecklistItemRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if checklistItemData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "checklist item not found")
		}

		taskChecklistData, err := s.TaskChecklistRepository.FindById(ctx, checklistItemData.TaskChecklistId)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
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

		var assignedMemberChecklist []*model.UserEntityModel
		if checklistItemData.AssignToUser != nil {
			assignToUserArr := general.StringToArrayInt(checklistItemData.AssignToUser)
			for _, v := range assignToUserArr {
				dataUser, err := s.UserRepository.FindById(ctx, v)
				if err != nil && err.Error() != "record not found" {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
				if dataUser != nil {
					assignedMemberChecklist = append(assignedMemberChecklist, dataUser)
				}
			}
		}

		var messageNotif []string
		newchecklistItemData := new(model.ChecklistItemEntityModel)
		newchecklistItemData.Context = ctx
		newchecklistItemData.ID = payload.ID
		if payload.TaskChecklistId != nil {
			newTaskChecklistData, err := s.TaskChecklistRepository.FindById(ctx, *payload.TaskChecklistId)
			if err != nil && err.Error() != "record not found" {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
			if newTaskChecklistData == nil {
				return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "task checklist not found")
			}
			newTaskData, err := s.TaskRepository.FindById(ctx, newTaskChecklistData.TaskId)
			if err != nil && err.Error() != "record not found" {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}

			newchecklistItemData.TaskChecklistId = newTaskChecklistData.ID
			if taskChecklistData.ID != newTaskChecklistData.ID {
				messageNotif = append(messageNotif, fmt.Sprintf("Item (%s) dipindahkan ke %s - %s", checklistItemData.Title, newTaskChecklistData.Title, newTaskData.Title))
			}
		}
		if payload.Title != nil {
			newchecklistItemData.Title = *payload.Title
			if checklistItemData.Title != newchecklistItemData.Title {
				messageNotif = append(messageNotif, fmt.Sprintf("Item (%s) berganti nama menjadi (%s)", checklistItemData.Title, newchecklistItemData.Title))
			}
		}
		if payload.AssignToUser != nil {
			strAssignToUser := general.ArrayIntToString(payload.AssignToUser)
			newchecklistItemData.AssignToUser = &strAssignToUser
			if strAssignToUser == "" || strAssignToUser == "0" {
				if err = s.ChecklistItemRepository.UpdateToNull(ctx, newchecklistItemData, "assign_to_user").Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
			added, removed := general.DiffIntSlices(general.StringToArrayInt(checklistItemData.AssignToUser), payload.AssignToUser)
			if added != nil {
				var userAddedArr []string
				var userAddedIdArr []int
				for _, v := range added {
					dataUser, err := s.UserRepository.FindById(ctx, v)
					if err != nil && err.Error() != "record not found" {
						return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
					}
					if dataUser != nil {
						userAddedArr = append(userAddedArr, dataUser.Name)
						userAddedIdArr = append(userAddedIdArr, dataUser.ID)
					}
				}
				for _, v := range userAddedIdArr {
					modelNotifikasi := new(model.NotifikasiEntityModel)
					modelNotifikasi.Context = ctx
					modelNotifikasi.Title = fmt.Sprintf("Anda telah ditambahkan ke item (%s) oleh %s", checklistItemData.Title, userLogin.Name)
					modelNotifikasi.Message = "Klik untuk melihat detail tugas"
					modelNotifikasi.IsRead = false
					modelNotifikasi.UserId = v
					modelNotifikasi.TaskId = taskData.ID
					if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
						return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
					}
				}
				messageNotif = append(messageNotif, fmt.Sprintf("User %s ditambahkan ke item (%s)", general.FormatNamesFromArray(userAddedArr), checklistItemData.Title))
			}
			if removed != nil {
				var userRemovedArr []string
				var userRemovedIdArr []int
				for _, v := range removed {
					dataUser, err := s.UserRepository.FindById(ctx, v)
					if err != nil && err.Error() != "record not found" {
						return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
					}
					if dataUser != nil {
						userRemovedArr = append(userRemovedArr, dataUser.Name)
						userRemovedIdArr = append(userRemovedIdArr, dataUser.ID)
					}
				}
				for _, v := range userRemovedIdArr {
					modelNotifikasi := new(model.NotifikasiEntityModel)
					modelNotifikasi.Context = ctx
					modelNotifikasi.Title = fmt.Sprintf("Anda telah dikeluarkan dari item (%s) oleh %s", checklistItemData.Title, userLogin.Name)
					modelNotifikasi.Message = "Hubungi administrator anda"
					modelNotifikasi.IsRead = false
					modelNotifikasi.UserId = v
					modelNotifikasi.TaskId = constant.BLANK_TASK_ID
					if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
						return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
					}
				}
				messageNotif = append(messageNotif, fmt.Sprintf("User %s dikeluarkan dari item (%s)", general.FormatNamesFromArray(userRemovedArr), checklistItemData.Title))
			}
		}
		if payload.IsCompleted != nil {
			newchecklistItemData.IsCompleted = *payload.IsCompleted
			if err = s.ChecklistItemRepository.UpdateCompleted(ctx, newchecklistItemData).Error; err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
			if *payload.IsCompleted {
				messageNotif = append(messageNotif, fmt.Sprintf("Item (%s) ditandai sebagai selesai", checklistItemData.Title))
			} else {
				messageNotif = append(messageNotif, fmt.Sprintf("Item (%s) ditandai sebagai belum selesai", checklistItemData.Title))
			}
		}
		if payload.DueDate != nil {
			if *payload.DueDate != "" {
				parsedDueDate, err := general.Parse("2006-01-02 15:04:05", *payload.DueDate)
				if err != nil {
					return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "err parse due date:"+err.Error())
				}
				newchecklistItemData.DueDate = &parsedDueDate
				if checklistItemData.DueDate != newchecklistItemData.DueDate {
					messageNotif = append(messageNotif, fmt.Sprintf("Tenggat waktu untuk item (%s) telah ditambahkan: %s", checklistItemData.Title, parsedDueDate))
				}
			} else {
				if err = s.ChecklistItemRepository.UpdateToNull(ctx, newchecklistItemData, "due_date").Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
				messageNotif = append(messageNotif, fmt.Sprintf("Tenggat waktu untuk item (%s) telah dihapus dari tugas", checklistItemData.Title))
			}
		}
		if payload.SortNumber != nil {
			newchecklistItemData.SortNumber = *payload.SortNumber
		}
		if err = s.ChecklistItemRepository.Update(ctx, newchecklistItemData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		newTaskChecklistData := new(model.TaskChecklistEntityModel)
		newTaskChecklistData.Context = ctx
		newTaskChecklistData.ID = checklistItemData.TaskChecklistId
		newTaskChecklistData.UpdatedAt = general.NowLocal()
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
					modelNotifikasi.Title = fmt.Sprintf("%s telah mengupdate item (%s) pada checklist (%s) di %s", userLogin.Name, checklistItemData.Title, taskChecklistData.Title, taskData.Title)
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
				modelNotifikasi.Title = fmt.Sprintf("Tugas yang anda buat (%s) terdapat item (%s) pada checklist (%s) yang telah diupdate oleh %s", taskData.Title, checklistItemData.Title, taskChecklistData.Title, userLogin.Name)
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
					modelNotifikasi.Title = fmt.Sprintf("%s telah mengupdate item (%s) pada checklist (%s) di %s", userLogin.Name, checklistItemData.Title, taskChecklistData.Title, taskData.Title)
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

		for _, v := range assignedMemberChecklist {
			alreadyNotif := false
			for _, j := range assignedMember {
				if v.ID == j.ID {
					alreadyNotif = true
				}
			}
			if alreadyNotif {
				continue
			}
			if v.Role.ID != constant.ROLE_ID_ADMIN && v.ID != taskData.CreateBy.ID {
				for _, d := range messageNotif {
					modelNotifikasi := new(model.NotifikasiEntityModel)
					modelNotifikasi.Context = ctx
					modelNotifikasi.Title = fmt.Sprintf("%s telah mengupdate item (%s) pada checklist (%s) di %s", userLogin.Name, checklistItemData.Title, taskChecklistData.Title, taskData.Title)
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

func (s *service) Delete(ctx *abstraction.Context, payload *dto.ChecklistItemDeleteByIDRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		checklistItemData, err := s.ChecklistItemRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if checklistItemData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "checklist item not found")
		}

		taskChecklistData, err := s.TaskChecklistRepository.FindById(ctx, checklistItemData.TaskChecklistId)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
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

		var assignedMemberChecklist []*model.UserEntityModel
		if checklistItemData.AssignToUser != nil {
			assignToUserArr := general.StringToArrayInt(checklistItemData.AssignToUser)
			for _, v := range assignToUserArr {
				dataUser, err := s.UserRepository.FindById(ctx, v)
				if err != nil && err.Error() != "record not found" {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
				if dataUser != nil {
					assignedMemberChecklist = append(assignedMemberChecklist, dataUser)
				}
			}
		}

		newChecklistItemData := new(model.ChecklistItemEntityModel)
		newChecklistItemData.Context = ctx
		newChecklistItemData.ID = payload.ID
		newChecklistItemData.IsDelete = true
		if err = s.ChecklistItemRepository.Update(ctx, newChecklistItemData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		newTaskChecklistData := new(model.TaskChecklistEntityModel)
		newTaskChecklistData.Context = ctx
		newTaskChecklistData.ID = checklistItemData.TaskChecklistId
		newTaskChecklistData.UpdatedAt = general.NowLocal()
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
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("Item (%s) pada checklist (%s) telah dihapus oleh %s", checklistItemData.Title, taskChecklistData.Title, userLogin.Name)
				modelNotifikasi.Message = fmt.Sprintf("Tugas: %s", taskData.Title)
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
			modelNotifikasi.Title = fmt.Sprintf("Pada tugas yang anda buat terdapat item (%s) di checklist (%s) yang telah dihapus oleh %s", checklistItemData.Title, taskChecklistData.Title, userLogin.Name)
			modelNotifikasi.Message = fmt.Sprintf("Tugas: %s", taskData.Title)
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
				modelNotifikasi.Title = fmt.Sprintf("Item (%s) pada checklist (%s) telah dihapus oleh %s", checklistItemData.Title, taskChecklistData.Title, userLogin.Name)
				modelNotifikasi.Message = fmt.Sprintf("Tugas: %s", taskData.Title)
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = v.ID
				modelNotifikasi.TaskId = taskData.ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
		}

		for _, v := range assignedMemberChecklist {
			alreadyNotif := false
			for _, j := range assignedMember {
				if v.ID == j.ID {
					alreadyNotif = true
				}
			}
			if alreadyNotif {
				continue
			}
			if v.Role.ID != constant.ROLE_ID_ADMIN && v.ID != taskData.CreateBy.ID {
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("Item (%s) pada checklist (%s) telah dihapus oleh %s", checklistItemData.Title, taskChecklistData.Title, userLogin.Name)
				modelNotifikasi.Message = fmt.Sprintf("Tugas: %s", taskData.Title)
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = v.ID
				modelNotifikasi.TaskId = taskData.ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
		}

		if err := s.TaskCommentRepository.CreateHistory(ctx, taskChecklistData.TaskId, fmt.Sprintf("Item (%s) pada checklist (%s) telah dihapus oleh %s", checklistItemData.Title, taskChecklistData.Title, userLogin.Name)).Error; err != nil {
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

func (s *service) ConvertToTask(ctx *abstraction.Context, payload *dto.ChecklistItemConvertToTaskRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		checklistItemData, err := s.ChecklistItemRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if checklistItemData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "checklist item not found")
		}

		taskChecklistData, err := s.TaskChecklistRepository.FindById(ctx, checklistItemData.TaskChecklistId)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		taskData, err := s.TaskRepository.FindById(ctx, taskChecklistData.TaskId)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		boardData, err := s.BoardRepository.FindById(ctx, taskData.BoardId)
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

		var assignedMemberChecklist []*model.UserEntityModel
		if checklistItemData.AssignToUser != nil {
			assignToUserArr := general.StringToArrayInt(checklistItemData.AssignToUser)
			for _, v := range assignToUserArr {
				dataUser, err := s.UserRepository.FindById(ctx, v)
				if err != nil && err.Error() != "record not found" {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
				if dataUser != nil {
					assignedMemberChecklist = append(assignedMemberChecklist, dataUser)
				}
			}
		}

		dataTaskInBoard, err := s.TaskRepository.FindByBoardId(ctx, taskData.BoardId)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		sortNum := 1
		if dataTaskInBoard != nil {
			sortNum = dataTaskInBoard.SortNumber + 1
		}

		modelTask := &model.TaskEntityModel{
			Context: ctx,
			TaskEntity: model.TaskEntity{
				BoardId:     taskData.BoardId,
				Title:       checklistItemData.Title,
				IsCompleted: false,
				IsDelete:    false,
				SortNumber:  sortNum,
			},
		}

		if checklistItemData.AssignToUser != nil {
			modelTask.AssignToUser = checklistItemData.AssignToUser
		}
		if checklistItemData.DueDate != nil {
			modelTask.DueDate = checklistItemData.DueDate
		}

		if err := s.TaskRepository.Create(ctx, modelTask).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		keyWatchTask := general.GenerateKeyWatchTask(ctx.Auth.ID, modelTask.ID)
		s.DbRedis.Set(context.Background(), keyWatchTask, strconv.FormatBool(false), 0)

		newBoardData := new(model.BoardEntityModel)
		newBoardData.Context = ctx
		newBoardData.ID = taskData.BoardId
		newBoardData.TaskTotal = boardData.TaskTotal + 1
		if err = s.BoardRepository.UpdateTaskTotalById(ctx, newBoardData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		for _, v := range userAdmin {
			if v.ID != ctx.Auth.ID {
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("Tugas baru telah dibuat oleh %s dari item (%s) di checklist %s", userLogin.Name, checklistItemData.Title, taskChecklistData.Title)
				modelNotifikasi.Message = fmt.Sprintf("Tugas: %s", taskData.Title)
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = v.ID
				modelNotifikasi.TaskId = modelTask.ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
		}

		if taskData.CreateBy.RoleId != constant.ROLE_ID_ADMIN {
			modelNotifikasi := new(model.NotifikasiEntityModel)
			modelNotifikasi.Context = ctx
			modelNotifikasi.Title = fmt.Sprintf("Pada tugas yang anda buat terdapat item (%s) di checklist (%s) yang telah dikonversi ke tugas oleh %s", checklistItemData.Title, taskChecklistData.Title, userLogin.Name)
			modelNotifikasi.Message = fmt.Sprintf("Tugas: %s", taskData.Title)
			modelNotifikasi.IsRead = false
			modelNotifikasi.UserId = taskData.CreateBy.ID
			modelNotifikasi.TaskId = modelTask.ID
			if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
		}

		for _, v := range assignedMember {
			if v.Role.ID != constant.ROLE_ID_ADMIN && v.ID != taskData.CreateBy.ID {
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("Tugas baru telah dibuat oleh %s dari item (%s) di checklist %s", userLogin.Name, checklistItemData.Title, taskChecklistData.Title)
				modelNotifikasi.Message = fmt.Sprintf("Tugas: %s", taskData.Title)
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = v.ID
				modelNotifikasi.TaskId = taskData.ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
		}

		for _, v := range assignedMemberChecklist {
			alreadyNotif := false
			for _, j := range assignedMember {
				if v.ID == j.ID {
					alreadyNotif = true
				}
			}
			if alreadyNotif {
				continue
			}
			if v.Role.ID != constant.ROLE_ID_ADMIN && v.ID != taskData.CreateBy.ID {
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("Tugas baru telah dibuat oleh %s dari item (%s) di checklist %s", userLogin.Name, checklistItemData.Title, taskChecklistData.Title)
				modelNotifikasi.Message = fmt.Sprintf("Tugas: %s", taskData.Title)
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = v.ID
				modelNotifikasi.TaskId = taskData.ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
		}

		if err := s.TaskCommentRepository.CreateHistory(ctx, modelTask.ID, fmt.Sprintf("Tugas dikonversi dari item (%s) di checklist (%s) pada tugas (%s) oleh %s", checklistItemData.Title, taskChecklistData.Title, taskData.Title, userLogin.Name)).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		newChecklistItemData := new(model.ChecklistItemEntityModel)
		newChecklistItemData.Context = ctx
		newChecklistItemData.ID = payload.ID
		newChecklistItemData.IsDelete = true
		if err = s.ChecklistItemRepository.Update(ctx, newChecklistItemData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		newTaskChecklistData := new(model.TaskChecklistEntityModel)
		newTaskChecklistData.Context = ctx
		newTaskChecklistData.ID = checklistItemData.TaskChecklistId
		newTaskChecklistData.UpdatedAt = general.NowLocal()
		if err = s.TaskChecklistRepository.Update(ctx, newTaskChecklistData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		if err := s.TaskCommentRepository.CreateHistory(ctx, taskChecklistData.TaskId, fmt.Sprintf("Item (%s) dari checklist (%s) telah dikonversi menjadi tugas oleh %s", checklistItemData.Title, taskChecklistData.Title, userLogin.Name)).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		return nil
	}); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message": "success convert!",
	}, nil
}
