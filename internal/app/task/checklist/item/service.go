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

		if payload.AssignToUser != nil {
			strAssignToUser := general.ArrayIntToString(payload.AssignToUser)
			modelChecklistItem.AssignToUser = &strAssignToUser
		}
		if payload.DueDate != nil {
			parsedDueDate, err := general.Parse("2006-01-02", *payload.DueDate)
			if err != nil {
				return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "err parse due date:"+err.Error())
			}
			modelChecklistItem.DueDate = &parsedDueDate
		}

		if err := s.ChecklistItemRepository.Create(ctx, modelChecklistItem).Error; err != nil {
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
			"created_at":     v.CreatedAt,
			"updated_at":     v.UpdatedAt,
		}

		if v.AssignToUser != nil {
			var assignToUser []map[string]interface{}
			assignToUserArr := general.StringToArrayInt(*v.AssignToUser)
			for _, v := range assignToUserArr {
				dataUser, err := s.UserRepository.FindById(ctx, v)
				if err != nil && err.Error() != "record not found" {
					return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
				if dataUser != nil {
					assignToUser = append(assignToUser, map[string]interface{}{
						"id":    dataUser.ID,
						"name":  dataUser.Name,
						"email": dataUser.Email,
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

		newchecklistItemData := new(model.ChecklistItemEntityModel)
		newchecklistItemData.Context = ctx
		newchecklistItemData.ID = payload.ID
		if payload.TaskChecklistId != nil {
			taskChecklistData, err := s.TaskChecklistRepository.FindById(ctx, *payload.TaskChecklistId)
			if err != nil && err.Error() != "record not found" {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
			if taskChecklistData == nil {
				return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "task checklist not found")
			}

			newchecklistItemData.TaskChecklistId = *payload.TaskChecklistId
		}
		if payload.Title != nil {
			newchecklistItemData.Title = *payload.Title
		}
		if payload.AssignToUser != nil {
			strAssignToUser := general.ArrayIntToString(payload.AssignToUser)
			newchecklistItemData.AssignToUser = &strAssignToUser
			if strAssignToUser == "0" {
				if err = s.ChecklistItemRepository.UpdateToNull(ctx, newchecklistItemData, "assign_to_user").Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
		}
		if payload.IsCompleted != nil {
			newchecklistItemData.IsCompleted = *payload.IsCompleted
			if err = s.ChecklistItemRepository.UpdateCompleted(ctx, newchecklistItemData).Error; err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
		}
		if payload.DueDate != nil {
			if *payload.DueDate != "" {
				parsedDueDate, err := general.Parse("2006-01-02", *payload.DueDate)
				if err != nil {
					return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "err parse due date:"+err.Error())
				}
				newchecklistItemData.DueDate = &parsedDueDate
			} else {
				if err = s.ChecklistItemRepository.UpdateToNull(ctx, newchecklistItemData, "due_date").Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
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
		userLogin, err := s.UserRepository.FindById(ctx, ctx.Auth.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		userAdmin, err := s.UserRepository.FindByRoleId(ctx, constant.ROLE_ID_ADMIN)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

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

		modelNotifikasi := new(model.NotifikasiEntityModel)
		modelNotifikasi.Context = ctx
		modelNotifikasi.Title = fmt.Sprintf("Tugas baru telah dibuat oleh %s dari item checklist %s", userLogin.Name, checklistItemData.Title)
		modelNotifikasi.Message = modelTask.Title
		modelNotifikasi.IsRead = false
		modelNotifikasi.UserId = userAdmin.ID
		modelNotifikasi.TaskId = modelTask.ID
		if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
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

		return nil
	}); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message": "success convert!",
	}, nil
}
