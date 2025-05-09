package task

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
	"selarashomeid/pkg/gdrive"
	"selarashomeid/pkg/util/general"
	"selarashomeid/pkg/util/response"
	"selarashomeid/pkg/util/trxmanager"
	"strconv"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/drive/v3"
	"gorm.io/gorm"
)

type Service interface {
	Create(ctx *abstraction.Context, payload *dto.TaskCreateRequest) (map[string]interface{}, error)
	Delete(ctx *abstraction.Context, payload *dto.TaskDeleteByIDRequest) (map[string]interface{}, error)
	FindByBoardId(ctx *abstraction.Context, payload *dto.TaskFindByBoardIDRequest) (map[string]interface{}, error)
	Update(ctx *abstraction.Context, payload *dto.TaskUpdateRequest) (map[string]interface{}, error)
	FindById(ctx *abstraction.Context, payload *dto.TaskFindByIDRequest) (map[string]interface{}, error)
	Find(ctx *abstraction.Context) (map[string]interface{}, error)
}

type service struct {
	TaskRepository          repository.Task
	BoardRepository         repository.Board
	WorkspaceRepository     repository.Workspace
	UserRepository          repository.User
	TaskFileRepository      repository.TaskFile
	TaskCommentRepository   repository.TaskComment
	NotifikasiRepository    repository.Notifikasi
	TaskLabelRepository     repository.TaskLabel
	TaskChecklistRepository repository.TaskChecklist
	ChecklistItemRepository repository.ChecklistItem

	DB      *gorm.DB
	DbRedis *redis.Client
	sDrive  *drive.Service
	fDrive  *drive.File
}

func NewService(f *factory.Factory) Service {
	return &service{
		TaskRepository:          f.TaskRepository,
		BoardRepository:         f.BoardRepository,
		WorkspaceRepository:     f.WorkspaceRepository,
		UserRepository:          f.UserRepository,
		TaskFileRepository:      f.TaskFileRepository,
		TaskCommentRepository:   f.TaskCommentRepository,
		NotifikasiRepository:    f.NotifikasiRepository,
		TaskLabelRepository:     f.TaskLabelRepository,
		TaskChecklistRepository: f.TaskChecklistRepository,
		ChecklistItemRepository: f.ChecklistItemRepository,

		DB:      f.Db,
		DbRedis: f.DbRedis,
		sDrive:  f.GDrive.Service,
		fDrive:  f.GDrive.FolderTaskCover,
	}
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.TaskCreateRequest) (map[string]interface{}, error) {
	returnId := 0
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		userLogin, err := s.UserRepository.FindById(ctx, ctx.Auth.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		userAdmin, err := s.UserRepository.FindByRoleIdArr(ctx, constant.ROLE_ID_ADMIN, true)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		boardData, err := s.BoardRepository.FindById(ctx, payload.BoardId)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if boardData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "board not found")
		}

		dataTaskInBoard, err := s.TaskRepository.FindByBoardId(ctx, payload.BoardId)
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
				BoardId:     payload.BoardId,
				Title:       payload.Title,
				IsCompleted: false,
				IsDelete:    false,
				SortNumber:  sortNum,
			},
		}

		if err := s.TaskRepository.Create(ctx, modelTask).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		keyWatchTask := general.GenerateKeyWatchTask(userLogin.ID, modelTask.ID)
		s.DbRedis.Set(context.Background(), keyWatchTask, strconv.FormatBool(false), 0)

		newBoardData := new(model.BoardEntityModel)
		newBoardData.Context = ctx
		newBoardData.ID = payload.BoardId
		newBoardData.TaskTotal = boardData.TaskTotal + 1
		if err = s.BoardRepository.UpdateTaskTotalById(ctx, newBoardData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		for _, v := range userAdmin {
			if v.ID != userLogin.ID {
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("Tugas baru telah dibuat oleh %s", userLogin.Name)
				modelNotifikasi.Message = modelTask.Title
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = v.ID
				modelNotifikasi.TaskId = modelTask.ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
		}

		msgHistory := fmt.Sprintf("Tugas dibuat oleh %s", userLogin.Name)
		if ctx.Auth.RoleID == constant.ROLE_ID_ADMIN {
			msgHistory = fmt.Sprintf("Tugas dibuat oleh admin (%s)", userLogin.Name)
		}
		if err := s.TaskCommentRepository.CreateHistory(ctx, modelTask.ID, msgHistory).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		returnId = modelTask.ID
		return nil
	}); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message": "success create!",
		"id":      returnId,
	}, nil
}

func (s *service) Delete(ctx *abstraction.Context, payload *dto.TaskDeleteByIDRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		taskData, err := s.TaskRepository.FindById(ctx, payload.ID)
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

		boardData, err := s.BoardRepository.FindById(ctx, taskData.BoardId)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if boardData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "board not found")
		}

		workspaceData, err := s.WorkspaceRepository.FindById(ctx, boardData.WorkspaceId)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		newTaskData := new(model.TaskEntityModel)
		newTaskData.Context = ctx
		newTaskData.ID = payload.ID
		newTaskData.IsDelete = true
		if err = s.TaskRepository.Update(ctx, newTaskData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		newBoardData := new(model.BoardEntityModel)
		newBoardData.Context = ctx
		newBoardData.ID = boardData.ID
		newBoardData.TaskTotal = boardData.TaskTotal - 1
		if err = s.BoardRepository.UpdateTaskTotalById(ctx, newBoardData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		for _, v := range userAdmin {
			if v.ID != userLogin.ID {
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("Tugas (%s) telah dihapus oleh %s", taskData.Title, userLogin.Name)
				modelNotifikasi.Message = fmt.Sprintf("(%s - %s)", boardData.Name, workspaceData.Name)
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = v.ID
				modelNotifikasi.TaskId = constant.BLANK_TASK_ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
		}

		if taskData.CreateBy.RoleId != constant.ROLE_ID_ADMIN {
			modelNotifikasi := new(model.NotifikasiEntityModel)
			modelNotifikasi.Context = ctx
			modelNotifikasi.Title = fmt.Sprintf("Tugas yang anda buat (%s) telah dihapus oleh %s", taskData.Title, userLogin.Name)
			modelNotifikasi.Message = fmt.Sprintf("(%s - %s)", boardData.Name, workspaceData.Name)
			modelNotifikasi.IsRead = false
			modelNotifikasi.UserId = taskData.CreateBy.ID
			modelNotifikasi.TaskId = constant.BLANK_TASK_ID
			if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
		}

		for _, v := range assignedMember {
			if v.Role.ID != constant.ROLE_ID_ADMIN && v.ID != taskData.CreateBy.ID {
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("Tugas (%s) telah dihapus oleh %s", taskData.Title, userLogin.Name)
				modelNotifikasi.Message = fmt.Sprintf("(%s - %s)", boardData.Name, workspaceData.Name)
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = v.ID
				modelNotifikasi.TaskId = constant.BLANK_TASK_ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
		}

		if err := s.TaskCommentRepository.CreateHistory(ctx, taskData.ID, fmt.Sprintf("Tugas dihapus oleh %s", userLogin.Name)).Error; err != nil {
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

func (s *service) FindByBoardId(ctx *abstraction.Context, payload *dto.TaskFindByBoardIDRequest) (map[string]interface{}, error) {
	var res []map[string]interface{} = nil

	boardData, err := s.BoardRepository.FindById(ctx, payload.BoardID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	if boardData == nil {
		return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "board not found")
	}

	data, err := s.TaskRepository.FindByBoardIdArr(ctx, payload.BoardID, false)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	count, err := s.TaskRepository.CountByBoardIdArr(ctx, payload.BoardID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}

	for _, v := range data {
		countFileData, err := s.TaskFileRepository.CountByTaskId(ctx, v.ID)
		if err != nil && err.Error() != "record not found" {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		countCommentData, err := s.TaskCommentRepository.CountCommentByTaskId(ctx, v.ID)
		if err != nil && err.Error() != "record not found" {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		countCompleted, countTotal, err := s.ChecklistItemRepository.CountByTaskId(ctx, v.ID)
		if err != nil && err.Error() != "record not found" {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		description := false
		if v.Description != nil && *v.Description != "" {
			description = true
		}

		isWatch := false
		keyWatchTask := general.GenerateKeyWatchTask(ctx.Auth.ID, v.ID)
		val, errGetKeyWatch := s.DbRedis.Get(context.Background(), keyWatchTask).Result()
		if errGetKeyWatch == redis.Nil {
			s.DbRedis.Set(context.Background(), keyWatchTask, strconv.FormatBool(false), 0)
		} else if errGetKeyWatch != nil {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		} else {
			valBool, _ := strconv.ParseBool(val)
			isWatch = valBool
		}

		task := map[string]interface{}{
			"id":             v.ID,
			"board_id":       v.BoardId,
			"title":          v.Title,
			"description":    description,
			"assign_to_user": v.AssignToUser,
			"label":          v.Label,
			"is_completed":   v.IsCompleted,
			"due_date":       v.DueDate,
			"cover":          v.Cover,
			"cover_name":     v.CoverName,
			"file":           countFileData,
			"comment":        countCommentData,
			"checklist":      nil,
			"is_delete":      v.IsDelete,
			"created_at":     general.FormatWithZWithoutChangingTime(v.CreatedAt),
			"updated_at":     general.FormatWithZWithoutChangingTime(*v.UpdatedAt),
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
			"watch":       isWatch,
			"sort_number": v.SortNumber,
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
			task["assign_to_user"] = map[string]interface{}{
				"count": len(assignToUserArr),
				"data":  assignToUser,
			}
		}

		if v.Cover != nil {
			cover, err := gdrive.GetFile(s.sDrive, *v.Cover)
			if err != nil {
				return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "cover not found")
			}
			task["cover"] = map[string]interface{}{
				"view":       "https://lh3.googleusercontent.com/d/" + *v.Cover,
				"view_saved": general.ConvertLinkToFileSaved(cover.WebContentLink),
				"content":    cover.WebContentLink,
				"ext":        cover.FileExtension,
				"name":       cover.Name,
				"id":         cover.Id,
			}
		}

		if v.Label != nil {
			var label []map[string]interface{}
			labelArr := general.StringToArrayInt(v.Label)
			for _, v := range labelArr {
				dataLabel, err := s.TaskLabelRepository.FindById(ctx, v)
				if err != nil && err.Error() != "record not found" {
					return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
				if dataLabel != nil {
					label = append(label, map[string]interface{}{
						"id":    dataLabel.ID,
						"title": dataLabel.Title,
						"color": dataLabel.Color,
					})
				}
			}
			task["label"] = map[string]interface{}{
				"count": len(labelArr),
				"data":  label,
			}
		}

		if *countTotal > 0 {
			task["checklist"] = fmt.Sprintf("%d/%d", *countCompleted, *countTotal)
		}

		res = append(res, task)
	}

	return map[string]interface{}{
		"count": count,
		"data":  res,
	}, nil
}

func (s *service) Update(ctx *abstraction.Context, payload *dto.TaskUpdateRequest) (map[string]interface{}, error) {
	var allFileUploaded []string = nil
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		changesTaskTotal := new(model.BoardChangesTaskTotal)
		taskData, err := s.TaskRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if taskData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "task not found")
		}
		changesTaskTotal.OldBoard = &taskData.BoardId

		boardData, err := s.BoardRepository.FindById(ctx, taskData.BoardId)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		workspaceData, err := s.WorkspaceRepository.FindById(ctx, boardData.WorkspaceId)
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
		newTaskData := new(model.TaskEntityModel)
		newTaskData.Context = ctx
		newTaskData.ID = payload.ID
		if payload.BoardId != nil {
			newBoardData, err := s.BoardRepository.FindById(ctx, *payload.BoardId)
			if err != nil && err.Error() != "record not found" {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
			if newBoardData == nil {
				return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "board not found")
			}
			changesTaskTotal.NewBoard = &newBoardData.ID

			newWorkspaceData, err := s.WorkspaceRepository.FindById(ctx, newBoardData.WorkspaceId)
			if err != nil && err.Error() != "record not found" {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}

			newTaskData.BoardId = *payload.BoardId
			if boardData.ID != newBoardData.ID {
				messageNotif = append(messageNotif, fmt.Sprintf("Tugas dipindahkan ke %s - %s", newBoardData.Name, newWorkspaceData.Name))
			}
		}
		if payload.Title != nil {
			newTaskData.Title = *payload.Title
			if taskData.Title != newTaskData.Title {
				messageNotif = append(messageNotif, fmt.Sprintf("Tugas berganti nama menjadi (%s)", newTaskData.Title))
			}
		}
		if payload.Description != nil {
			newTaskData.Description = payload.Description
			if *payload.Description == "" {
				if err = s.TaskRepository.UpdateToNull(ctx, newTaskData, "description").Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
			if taskData.Description != newTaskData.Description {
				messageNotif = append(messageNotif, fmt.Sprintf("Deskripsi tugas diperbarui: %s", *payload.Description))
			}
		}
		if payload.AssignToUser != nil {
			strAssignToUser := general.ArrayIntToString(payload.AssignToUser)
			newTaskData.AssignToUser = &strAssignToUser
			if strAssignToUser == "" || strAssignToUser == "0" {
				if err = s.TaskRepository.UpdateToNull(ctx, newTaskData, "assign_to_user").Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
			added, removed := general.DiffIntSlices(general.StringToArrayInt(taskData.AssignToUser), payload.AssignToUser)
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
					modelNotifikasi.Title = fmt.Sprintf("Anda telah ditambahkan ke tugas (%s) oleh %s", taskData.Title, userLogin.Name)
					modelNotifikasi.Message = "Klik untuk melihat detail tugas"
					modelNotifikasi.IsRead = false
					modelNotifikasi.UserId = v
					modelNotifikasi.TaskId = taskData.ID
					if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
						return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
					}
				}
				messageNotif = append(messageNotif, fmt.Sprintf("User %s ditambahkan ke tugas", general.FormatNamesFromArray(userAddedArr)))
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
					modelNotifikasi.Title = fmt.Sprintf("Anda telah dikeluarkan dari tugas (%s) oleh %s", taskData.Title, userLogin.Name)
					modelNotifikasi.Message = "Hubungi administrator anda"
					modelNotifikasi.IsRead = false
					modelNotifikasi.UserId = v
					modelNotifikasi.TaskId = constant.BLANK_TASK_ID
					if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
						return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
					}
				}
				messageNotif = append(messageNotif, fmt.Sprintf("User %s dikeluarkan dari tugas", general.FormatNamesFromArray(userRemovedArr)))
			}
		}
		if payload.Label != nil {
			strLabel := general.ArrayIntToString(payload.Label)
			newTaskData.Label = &strLabel
			if strLabel == "" || strLabel == "0" {
				if err = s.TaskRepository.UpdateToNull(ctx, newTaskData, "label").Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
			added, removed := general.DiffIntSlices(general.StringToArrayInt(taskData.Label), payload.Label)
			if added != nil {
				var labelAddedArr []string
				for _, v := range added {
					dataLabel, err := s.TaskLabelRepository.FindById(ctx, v)
					if err != nil && err.Error() != "record not found" {
						return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
					}
					if dataLabel != nil {
						labelAddedArr = append(labelAddedArr, dataLabel.Title)
					}
				}
				messageNotif = append(messageNotif, fmt.Sprintf("Label %s ditambahkan ke tugas", general.FormatNamesFromArray(labelAddedArr)))
			}
			if removed != nil {
				var labelRemovedArr []string
				for _, v := range removed {
					dataLabel, err := s.TaskLabelRepository.FindById(ctx, v)
					if err != nil && err.Error() != "record not found" {
						return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
					}
					if dataLabel != nil {
						labelRemovedArr = append(labelRemovedArr, dataLabel.Title)
					}
				}
				messageNotif = append(messageNotif, fmt.Sprintf("Label %s dihapus dari tugas", general.FormatNamesFromArray(labelRemovedArr)))
			}
		}
		if payload.IsCompleted != nil {
			newTaskData.IsCompleted = *payload.IsCompleted
			if err = s.TaskRepository.UpdateCompleted(ctx, newTaskData).Error; err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
			if *payload.IsCompleted {
				messageNotif = append(messageNotif, "Tugas ditandai sebagai selesai")
			} else {
				messageNotif = append(messageNotif, "Tugas ditandai sebagai belum selesai")
			}
		}
		if payload.DueDate != nil {
			if *payload.DueDate != "" {
				parsedDueDate, err := general.Parse("2006-01-02 15:04:05", *payload.DueDate)
				if err != nil {
					return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "err parse due date:"+err.Error())
				}
				newTaskData.DueDate = &parsedDueDate
				if taskData.DueDate != newTaskData.DueDate {
					messageNotif = append(messageNotif, fmt.Sprintf("Tenggat waktu telah ditambahkan: %s", parsedDueDate))
				}
			} else {
				if err = s.TaskRepository.UpdateToNull(ctx, newTaskData, "due_date").Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
				messageNotif = append(messageNotif, "Tenggat waktu telah dihapus dari tugas")
			}
		}
		if payload.Cover != nil {
			file := payload.Cover[0]

			f, err := file.Open()
			if err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
			defer f.Close()

			isImageFile, fullFileName := general.ValidateImage(file.Filename)
			if !isImageFile {
				return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), fmt.Sprintf("file format for %s is not approved", file.Filename))
			}

			newFile, err := gdrive.CreateFile(s.sDrive, fullFileName, "application/octet-stream", f, s.fDrive.Id)
			if err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
			allFileUploaded = append(allFileUploaded, newFile.Id)

			newTaskData.Cover = &newFile.Id
			newTaskData.CoverName = &newFile.Name

			if taskData.Cover != nil {
				err = gdrive.DeleteFile(s.sDrive, *taskData.Cover)
				if err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
			if taskData.Cover != newTaskData.Cover {
				messageNotif = append(messageNotif, "Cover telah ditambahkan")
			}
		} else {
			if payload.DeleteCover != nil && *payload.DeleteCover {
				errDel := gdrive.DeleteFile(s.sDrive, *taskData.Cover)
				if errDel != nil {
					logrus.Error("error delete file for cover:", errDel.Error())
				}
				if err = s.TaskRepository.UpdateToNull(ctx, newTaskData, "cover").Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
				if err = s.TaskRepository.UpdateToNull(ctx, newTaskData, "cover_name").Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
				messageNotif = append(messageNotif, "Cover telah dihapus")
			}
		}
		if payload.Watch != nil {
			keyWatchTask := general.GenerateKeyWatchTask(userLogin.ID, newTaskData.ID)
			s.DbRedis.Set(context.Background(), keyWatchTask, strconv.FormatBool(*payload.Watch), 0)
			if *payload.Watch {
				if userLogin.ID != taskData.CreateBy.ID {
					modelNotifikasi := new(model.NotifikasiEntityModel)
					modelNotifikasi.Context = ctx
					modelNotifikasi.Title = fmt.Sprintf("Tugas yang anda buat telah dilihat oleh %s", userLogin.Name)
					modelNotifikasi.Message = "Klik untuk melihat detail tugas"
					modelNotifikasi.IsRead = false
					modelNotifikasi.UserId = taskData.CreateBy.ID
					modelNotifikasi.TaskId = taskData.ID
					if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
						return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
					}
				}
			}
		}
		if payload.SortNumber != nil {
			newTaskData.SortNumber = *payload.SortNumber
		}
		if err = s.TaskRepository.Update(ctx, newTaskData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		if payload.BoardId != nil {
			boardDataNew, err := s.BoardRepository.FindById(ctx, *changesTaskTotal.NewBoard)
			if err != nil && err.Error() != "record not found" {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}

			newBoardDataNew := new(model.BoardEntityModel)
			newBoardDataNew.Context = ctx
			newBoardDataNew.ID = boardDataNew.ID
			newBoardDataNew.TaskTotal = boardDataNew.TaskTotal + 1
			if err = s.BoardRepository.UpdateTaskTotalById(ctx, newBoardDataNew).Error; err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}

			boardDataOld, err := s.BoardRepository.FindById(ctx, *changesTaskTotal.OldBoard)
			if err != nil && err.Error() != "record not found" {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}

			newBoardDataOld := new(model.BoardEntityModel)
			newBoardDataOld.Context = ctx
			newBoardDataOld.ID = boardDataOld.ID
			newBoardDataOld.TaskTotal = boardDataOld.TaskTotal - 1
			if err = s.BoardRepository.UpdateTaskTotalById(ctx, newBoardDataOld).Error; err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
		}

		for _, v := range userAdmin {
			if v.ID != userLogin.ID {
				for _, d := range messageNotif {
					modelNotifikasi := new(model.NotifikasiEntityModel)
					modelNotifikasi.Context = ctx
					modelNotifikasi.Title = fmt.Sprintf("%s telah mengupdate tugas (%s) di %s", userLogin.Name, taskData.Title, workspaceData.Name)
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
				modelNotifikasi.Title = fmt.Sprintf("Tugas yang anda buat (%s) telah diupdate oleh %s", taskData.Title, userLogin.Name)
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
					modelNotifikasi.Title = fmt.Sprintf("%s telah mengupdate tugas (%s) di %s", userLogin.Name, taskData.Title, workspaceData.Name)
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
		for _, v := range allFileUploaded {
			errDel := gdrive.DeleteFile(s.sDrive, v)
			if errDel != nil {
				logrus.Error("error delete file for error trxmanager:", errDel.Error())
			}
		}
		return nil, err
	}
	return map[string]interface{}{
		"message": "success update!",
	}, nil
}

func (s *service) FindById(ctx *abstraction.Context, payload *dto.TaskFindByIDRequest) (map[string]interface{}, error) {
	var res map[string]interface{} = nil

	data, err := s.TaskRepository.FindById(ctx, payload.ID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	fileData, err := s.TaskFileRepository.FindByTaskId(ctx, payload.ID, true)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	commentData, err := s.TaskCommentRepository.FindByTaskId(ctx, payload.ID, true)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	checklistData, err := s.TaskChecklistRepository.FindByTaskId(ctx, payload.ID, true)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	if data != nil {
		boardData, err := s.BoardRepository.FindById(ctx, data.BoardId)
		if err != nil && err.Error() != "record not found" {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		workspaceData, err := s.WorkspaceRepository.FindById(ctx, boardData.WorkspaceId)
		if err != nil && err.Error() != "record not found" {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		isWatch := false
		keyWatchTask := general.GenerateKeyWatchTask(ctx.Auth.ID, data.ID)
		val, errGetKeyWatch := s.DbRedis.Get(context.Background(), keyWatchTask).Result()
		if errGetKeyWatch == redis.Nil {
			s.DbRedis.Set(context.Background(), keyWatchTask, strconv.FormatBool(false), 0)
		} else if errGetKeyWatch != nil {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		} else {
			valBool, _ := strconv.ParseBool(val)
			isWatch = valBool
		}

		res = map[string]interface{}{
			"id":             data.ID,
			"board_id":       data.BoardId,
			"title":          data.Title,
			"description":    data.Description,
			"assign_to_user": data.AssignToUser,
			"label":          data.Label,
			"is_completed":   data.IsCompleted,
			"due_date":       data.DueDate,
			"cover":          data.Cover,
			"cover_name":     data.CoverName,
			"file":           nil,
			"comment":        nil,
			"checklist":      nil,
			"is_delete":      data.IsDelete,
			"created_at":     general.FormatWithZWithoutChangingTime(data.CreatedAt),
			"updated_at":     general.FormatWithZWithoutChangingTime(*data.UpdatedAt),
			"created_by": map[string]interface{}{
				"id":    data.CreateBy.ID,
				"name":  data.CreateBy.Name,
				"email": data.CreateBy.Email,
			},
			"updated_by": map[string]interface{}{
				"id":    data.UpdateBy.ID,
				"name":  data.UpdateBy.Name,
				"email": data.UpdateBy.Email,
			},
			"watch":       isWatch,
			"sort_number": data.SortNumber,
			"workspace": map[string]interface{}{
				"id":         workspaceData.ID,
				"name":       workspaceData.Name,
				"project_id": workspaceData.ProjectId,
			},
		}

		if data.AssignToUser != nil {
			var assignToUser []map[string]interface{}
			assignToUserArr := general.StringToArrayInt(data.AssignToUser)
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
			res["assign_to_user"] = map[string]interface{}{
				"count": len(assignToUserArr),
				"data":  assignToUser,
			}
		}

		if data.Cover != nil {
			cover, err := gdrive.GetFile(s.sDrive, *data.Cover)
			if err != nil {
				return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "cover not found")
			}
			res["cover"] = map[string]interface{}{
				"view":       "https://lh3.googleusercontent.com/d/" + *data.Cover,
				"view_saved": general.ConvertLinkToFileSaved(cover.WebContentLink),
				"content":    cover.WebContentLink,
				"ext":        cover.FileExtension,
				"name":       cover.Name,
				"id":         cover.Id,
			}
		}

		if data.Label != nil {
			var label []map[string]interface{}
			labelArr := general.StringToArrayInt(data.Label)
			for _, v := range labelArr {
				dataLabel, err := s.TaskLabelRepository.FindById(ctx, v)
				if err != nil && err.Error() != "record not found" {
					return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
				if dataLabel != nil {
					label = append(label, map[string]interface{}{
						"id":    dataLabel.ID,
						"title": dataLabel.Title,
						"color": dataLabel.Color,
					})
				}
			}
			res["label"] = map[string]interface{}{
				"count": len(labelArr),
				"data":  label,
			}
		}

		var resFile []map[string]interface{}
		for _, v := range fileData {
			fileDrive, err := gdrive.GetFile(s.sDrive, v.File)
			if err != nil {
				return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "file not found")
			}
			resFile = append(resFile, map[string]interface{}{
				"id":      v.ID,
				"task_id": v.TaskId,
				"file": map[string]interface{}{
					"view":       "https://lh3.googleusercontent.com/d/" + v.File,
					"view_saved": general.ConvertLinkToFileSaved(fileDrive.WebContentLink),
					"content":    fileDrive.WebContentLink,
					"ext":        fileDrive.FileExtension,
					"name":       fileDrive.Name,
				},
				"file_name":  v.FileName,
				"is_delete":  v.IsDelete,
				"created_at": general.FormatWithZWithoutChangingTime(v.CreatedAt),
				"updated_at": general.FormatWithZWithoutChangingTime(*v.UpdatedAt),
			})
		}
		res["file"] = map[string]interface{}{
			"count": len(fileData),
			"data":  resFile,
		}

		var resComment []map[string]interface{}
		for _, v := range commentData {
			resComment = append(resComment, map[string]interface{}{
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
		res["comment"] = map[string]interface{}{
			"count": len(commentData),
			"data":  resComment,
		}

		var resChecklist []map[string]interface{}
		for _, v := range checklistData {
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

			resChecklist = append(resChecklist, map[string]interface{}{
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
		res["checklist"] = map[string]interface{}{
			"count": len(checklistData),
			"data":  resChecklist,
		}

	}

	return map[string]interface{}{
		"data": res,
	}, nil
}

func (s *service) Find(ctx *abstraction.Context) (map[string]interface{}, error) {
	var (
		data  []*model.TaskEntityModel
		count *int
		err   error
		res   []map[string]interface{} = nil
	)

	data, err = s.TaskRepository.Find(ctx, false)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	count, err = s.TaskRepository.Count(ctx)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}

	for _, v := range data {
		fileData, err := s.TaskFileRepository.FindByTaskId(ctx, v.ID, true)
		if err != nil && err.Error() != "record not found" {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		commentData, err := s.TaskCommentRepository.FindCommentByTaskId(ctx, v.ID, true)
		if err != nil && err.Error() != "record not found" {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		countCompleted, countTotal, err := s.ChecklistItemRepository.CountByTaskId(ctx, v.ID)
		if err != nil && err.Error() != "record not found" {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		description := false
		if v.Description != nil && *v.Description != "" {
			description = true
		}

		isWatch := false
		keyWatchTask := general.GenerateKeyWatchTask(ctx.Auth.ID, v.ID)
		val, errGetKeyWatch := s.DbRedis.Get(context.Background(), keyWatchTask).Result()
		if errGetKeyWatch == redis.Nil {
			s.DbRedis.Set(context.Background(), keyWatchTask, strconv.FormatBool(false), 0)
		} else if errGetKeyWatch != nil {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		} else {
			valBool, _ := strconv.ParseBool(val)
			isWatch = valBool
		}

		task := map[string]interface{}{
			"id":             v.ID,
			"board_id":       v.BoardId,
			"title":          v.Title,
			"description":    description,
			"assign_to_user": v.AssignToUser,
			"label":          v.Label,
			"is_completed":   v.IsCompleted,
			"due_date":       v.DueDate,
			"cover":          v.Cover,
			"cover_name":     v.CoverName,
			"file":           len(fileData),
			"comment":        len(commentData),
			"checklist":      nil,
			"is_delete":      v.IsDelete,
			"created_at":     general.FormatWithZWithoutChangingTime(v.CreatedAt),
			"updated_at":     general.FormatWithZWithoutChangingTime(*v.UpdatedAt),
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
			"watch":       isWatch,
			"sort_number": v.SortNumber,
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
			task["assign_to_user"] = map[string]interface{}{
				"count": len(assignToUserArr),
				"data":  assignToUser,
			}
		}

		if v.Cover != nil {
			cover, err := gdrive.GetFile(s.sDrive, *v.Cover)
			if err != nil {
				return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "cover not found")
			}
			task["cover"] = map[string]interface{}{
				"view":       "https://lh3.googleusercontent.com/d/" + *v.Cover,
				"view_saved": general.ConvertLinkToFileSaved(cover.WebContentLink),
				"content":    cover.WebContentLink,
				"ext":        cover.FileExtension,
				"name":       cover.Name,
				"id":         cover.Id,
			}
		}

		if v.Label != nil {
			var label []map[string]interface{}
			labelArr := general.StringToArrayInt(v.Label)
			for _, v := range labelArr {
				dataLabel, err := s.TaskLabelRepository.FindById(ctx, v)
				if err != nil && err.Error() != "record not found" {
					return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
				if dataLabel != nil {
					label = append(label, map[string]interface{}{
						"id":    dataLabel.ID,
						"title": dataLabel.Title,
						"color": dataLabel.Color,
					})
				}
			}
			task["label"] = map[string]interface{}{
				"count": len(labelArr),
				"data":  label,
			}
		}

		if *countTotal > 0 {
			task["checklist"] = fmt.Sprintf("%d/%d", *countCompleted, *countTotal)
		}

		res = append(res, task)
	}
	return map[string]interface{}{
		"count": count,
		"data":  res,
	}, nil
}
