package task

import (
	"bytes"
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
	"selarashomeid/pkg/ws"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"github.com/xuri/excelize/v2"
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
	Export(ctx *abstraction.Context, payload *dto.TaskExportRequest) (string, *bytes.Buffer, error)
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
	ProjectRepository       repository.Project

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
		ProjectRepository:       f.ProjectRepository,

		DB:      f.Db,
		DbRedis: f.DbRedis,
		sDrive:  f.GDrive.Service,
		fDrive:  f.GDrive.FolderTaskCover,
	}
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.TaskCreateRequest) (map[string]interface{}, error) {
	returnId := 0
	var sendNotifTo []int = nil
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
				sendNotifTo = append(sendNotifTo, v.ID)
			}
		}

		msgHistory := fmt.Sprintf("Tugas dibuat oleh %s", userLogin.Name)
		if ctx.Auth.RoleID == constant.ROLE_ID_ADMIN {
			msgHistory = fmt.Sprintf("Tugas dibuat oleh admin (%s)", userLogin.Name)
		}
		if err := s.TaskCommentRepository.CreateHistory(ctx, modelTask.ID, msgHistory).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		keyWatchTask := general.GenerateKeyWatchTask(userLogin.ID, modelTask.ID)
		s.DbRedis.Set(context.Background(), keyWatchTask, strconv.FormatBool(false), 0)

		returnId = modelTask.ID
		return nil
	}); err != nil {
		return nil, err
	}

	for _, v := range general.RemoveDuplicateArrayInt(sendNotifTo) {
		if err := ws.PublishNotificationWithoutTransaction(v, s.DB, ctx); err != nil {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
	}

	return map[string]interface{}{
		"message": "success create!",
		"id":      returnId,
	}, nil
}

func (s *service) Delete(ctx *abstraction.Context, payload *dto.TaskDeleteByIDRequest) (map[string]interface{}, error) {
	var sendNotifTo []int = nil
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
				sendNotifTo = append(sendNotifTo, v.ID)
			}
		}

		if taskData.CreateBy.RoleId != constant.ROLE_ID_ADMIN && taskData.CreateBy.ID != userLogin.ID {
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
			sendNotifTo = append(sendNotifTo, taskData.CreateBy.ID)
		}

		for _, v := range assignedMember {
			if v.Role.ID != constant.ROLE_ID_ADMIN && v.ID != taskData.CreateBy.ID && v.ID != userLogin.ID {
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
				sendNotifTo = append(sendNotifTo, v.ID)
			}
		}

		if err := s.TaskCommentRepository.CreateHistory(ctx, taskData.ID, fmt.Sprintf("Tugas dihapus oleh %s", userLogin.Name)).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		return nil
	}); err != nil {
		return nil, err
	}

	for _, v := range general.RemoveDuplicateArrayInt(sendNotifTo) {
		if err := ws.PublishNotificationWithoutTransaction(v, s.DB, ctx); err != nil {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
	}

	return map[string]interface{}{
		"message": "success delete!",
	}, nil
}

func (s *service) FindByBoardId(ctx *abstraction.Context, payload *dto.TaskFindByBoardIDRequest) (map[string]interface{}, error) {
	var res []map[string]interface{} = nil

	userLogin, err := s.UserRepository.FindById(ctx, ctx.Auth.ID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}

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
		canAccess := false
		var membersCanAccess []int

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
					membersCanAccess = append(membersCanAccess, dataUser.ID)
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
				"view_saved": general.ConvertLinkToFileSaved(cover.WebContentLink, cover.Name, cover.FileExtension),
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

		if userLogin.RoleId == constant.ROLE_ID_ADMIN {
			canAccess = true
		}
		if userLogin.ID == v.CreatedBy {
			canAccess = true
		}
		if slices.Contains(membersCanAccess, userLogin.ID) {
			canAccess = true
		}
		task["can_access"] = canAccess

		res = append(res, task)
	}

	return map[string]interface{}{
		"count": count,
		"data":  res,
	}, nil
}

func (s *service) Update(ctx *abstraction.Context, payload *dto.TaskUpdateRequest) (map[string]interface{}, error) {
	var allFileUploaded []string = nil
	var sendNotifTo []int = nil
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
					if v != userLogin.ID {
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
						sendNotifTo = append(sendNotifTo, v)
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
					if v != userLogin.ID {
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
						sendNotifTo = append(sendNotifTo, v)
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
					sendNotifTo = append(sendNotifTo, taskData.CreateBy.ID)
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
					sendNotifTo = append(sendNotifTo, v.ID)
				}
			}
		}

		if taskData.CreateBy.RoleId != constant.ROLE_ID_ADMIN && taskData.CreateBy.ID != userLogin.ID {
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
				sendNotifTo = append(sendNotifTo, taskData.CreateBy.ID)
			}
		}

		for _, v := range assignedMember {
			if v.Role.ID != constant.ROLE_ID_ADMIN && v.ID != taskData.CreateBy.ID && v.ID != userLogin.ID {
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
					sendNotifTo = append(sendNotifTo, v.ID)
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

	for _, v := range general.RemoveDuplicateArrayInt(sendNotifTo) {
		if err := ws.PublishNotificationWithoutTransaction(v, s.DB, ctx); err != nil {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
	}

	return map[string]interface{}{
		"message": "success update!",
	}, nil
}

func (s *service) FindById(ctx *abstraction.Context, payload *dto.TaskFindByIDRequest) (map[string]interface{}, error) {
	var res map[string]interface{} = nil

	canAccess := false
	var membersCanAccess []int
	userLogin, err := s.UserRepository.FindById(ctx, ctx.Auth.ID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}

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
					membersCanAccess = append(membersCanAccess, dataUser.ID)
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
				"view_saved": general.ConvertLinkToFileSaved(cover.WebContentLink, cover.Name, cover.FileExtension),
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
					"view_saved": general.ConvertLinkToFileSaved(fileDrive.WebContentLink, fileDrive.Name, fileDrive.FileExtension),
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

		if userLogin.RoleId == constant.ROLE_ID_ADMIN {
			canAccess = true
		}
		if userLogin.ID == data.CreatedBy {
			canAccess = true
		}
		if slices.Contains(membersCanAccess, userLogin.ID) {
			canAccess = true
		}
		res["can_access"] = canAccess
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
				"view_saved": general.ConvertLinkToFileSaved(cover.WebContentLink, cover.Name, cover.FileExtension),
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

func (s *service) Export(ctx *abstraction.Context, payload *dto.TaskExportRequest) (string, *bytes.Buffer, error) {
	var (
		fileName string
		buf      bytes.Buffer
	)
	f := excelize.NewFile()
	if payload.WorkspaceId != nil {
		workspace, err := s.WorkspaceRepository.FindById(ctx, *payload.WorkspaceId)
		if err != nil && err.Error() != "record not found" {
			return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		project, err := s.ProjectRepository.FindById(ctx, workspace.ProjectId)
		if err != nil && err.Error() != "record not found" {
			return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		dataBoard, err := s.BoardRepository.FindByWorkspaceIdArr(ctx, workspace.ID, true)
		if err != nil && err.Error() != "record not found" {
			return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		sheetProject := "Project Info"
		ProcessProjectToExcel(s, ctx, f, sheetProject, nil, project, dataBoard) // process project to excel
		if err != nil {
			return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		for _, v := range dataBoard {
			board, err := s.BoardRepository.FindById(ctx, v.ID)
			if err != nil && err.Error() != "record not found" {
				return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
			dataTask, err := s.TaskRepository.FindByBoardIdArr(ctx, v.ID, true)
			if err != nil && err.Error() != "record not found" {
				return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}

			sheetBoard := fmt.Sprintf("%s (%d)", board.Name, board.TaskTotal)
			err = ProcessTaskToExcel(s, ctx, f, sheetBoard, dataTask) // process task to excel
			if err != nil {
				return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
		}

		if err := f.Write(&buf); err != nil {
			return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		fileName = fmt.Sprintf("%s (%s).xlsx", workspace.Name, general.NowLocal().Format("2006-01-02"))
	} else if payload.BoardId != nil {
		board, err := s.BoardRepository.FindById(ctx, *payload.BoardId)
		if err != nil && err.Error() != "record not found" {
			return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		workspace, err := s.WorkspaceRepository.FindById(ctx, board.WorkspaceId)
		if err != nil && err.Error() != "record not found" {
			return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		dataTask, err := s.TaskRepository.FindByBoardIdArr(ctx, board.ID, true)
		if err != nil && err.Error() != "record not found" {
			return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		sheetBoard := fmt.Sprintf("%s (%d)", board.Name, board.TaskTotal)
		err = ProcessTaskToExcel(s, ctx, f, sheetBoard, dataTask) // process task to excel
		if err != nil {
			return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		if err := f.Write(&buf); err != nil {
			return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		fileName = fmt.Sprintf("%s - %s (%s).xlsx", board.Name, workspace.Name, general.NowLocal().Format("2006-01-02"))
	} else {
		dataWorkspace, err := s.WorkspaceRepository.Find(ctx, true)
		if err != nil && err.Error() != "record not found" {
			return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		for _, v := range dataWorkspace {
			workspace, err := s.WorkspaceRepository.FindById(ctx, v.ID)
			if err != nil && err.Error() != "record not found" {
				return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
			project, err := s.ProjectRepository.FindById(ctx, workspace.ProjectId)
			if err != nil && err.Error() != "record not found" {
				return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
			dataBoard, err := s.BoardRepository.FindByWorkspaceIdArr(ctx, workspace.ID, true)
			if err != nil && err.Error() != "record not found" {
				return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}

			sheetProject := workspace.Name
			ProcessProjectToExcel(s, ctx, f, sheetProject, workspace, project, dataBoard) // process project to excel
			if err != nil {
				return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}

			for _, v := range dataBoard {
				board, err := s.BoardRepository.FindById(ctx, v.ID)
				if err != nil && err.Error() != "record not found" {
					return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
				dataTask, err := s.TaskRepository.FindByBoardIdArr(ctx, v.ID, true)
				if err != nil && err.Error() != "record not found" {
					return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}

				sheetBoard := fmt.Sprintf("%s - %s (%d)", general.GenerateInitial(workspace.Name), board.Name, board.TaskTotal)
				err = ProcessTaskToExcel(s, ctx, f, sheetBoard, dataTask) // process task to excel
				if err != nil {
					return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
		}
		if err := f.Write(&buf); err != nil {
			return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		fileName = fmt.Sprintf("All Data SelarasHomeId (%s).xlsx", general.NowLocal().Format("2006-01-02"))
	}

	return fileName, &buf, nil
}

func ProcessProjectToExcel(s *service, ctx *abstraction.Context, f *excelize.File, sheetName string, workspace *model.WorkspaceEntityModel, project *model.ProjectEntityModel, dataBoard []*model.BoardEntityModel) error {
	index, err := f.NewSheet(general.TruncateSheetName(sheetName))
	if err != nil {
		return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	f.DeleteSheet("Sheet1")
	f.SetActiveSheet(index)
	f.SetCellValue(sheetName, "A1", "Nama Proyek")
	f.SetCellValue(sheetName, "B1", "Lokasi")
	f.SetCellValue(sheetName, "C1", "Cover Proyek")
	f.SetCellValue(sheetName, "D1", "Tanggal Dibuat")
	f.SetCellValue(sheetName, "A2", project.Name)
	if project.Location != nil {
		f.SetCellValue(sheetName, "B2", *project.Location)
		f.SetCellHyperLink(sheetName, "B2", general.IsValidURL(*project.Location), "External")
	} else {
		f.SetCellValue(sheetName, "B2", "-")
	}
	if project.Cover != nil {
		cover, err := gdrive.GetFile(s.sDrive, *project.Cover)
		if err != nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "cover not found")
		}
		f.SetCellValue(sheetName, "C2", cover.WebContentLink)
		f.SetCellHyperLink(sheetName, "C2", general.IsValidURL(cover.WebContentLink), "External")
	} else {
		f.SetCellValue(sheetName, "C2", "-")
	}
	f.SetCellValue(sheetName, "D2", project.CreatedAt.Format("2006-01-02 15:04:05"))

	boardAvail := false
	for i, v := range dataBoard {
		boardAvail = true
		colA := fmt.Sprintf("A%d", i+5)
		f.SetCellValue(sheetName, colA, v.Name)
		if workspace != nil {
			f.SetCellHyperLink(sheetName, colA, fmt.Sprintf("#'%s - %s (%d)'!A1", general.GenerateInitial(workspace.Name), v.Name, v.TaskTotal), "Location")
		} else {
			f.SetCellHyperLink(sheetName, colA, fmt.Sprintf("#'%s (%d)'!A1", v.Name, v.TaskTotal), "Location")
		}
	}
	if boardAvail {
		f.SetCellValue(sheetName, "A4", "Board tersedia (klik untuk melihat):")
	} else {
		f.SetCellValue(sheetName, "A4", "Board tidak tersedia")
	}

	return nil
}

func ProcessTaskToExcel(s *service, ctx *abstraction.Context, f *excelize.File, sheetName string, dataTask []*model.TaskEntityModel) error {
	sheetBoard := sheetName
	index, err := f.NewSheet(general.TruncateSheetName(sheetBoard))
	if err != nil {
		return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	f.DeleteSheet("Sheet1")
	f.SetActiveSheet(index)
	f.SetCellValue(sheetBoard, "A1", "No")
	f.SetCellValue(sheetBoard, "B1", "Tugas")
	f.SetCellValue(sheetBoard, "C1", "Deskripsi")
	f.SetCellValue(sheetBoard, "D1", "Member")
	f.SetCellValue(sheetBoard, "E1", "Label")
	f.SetCellValue(sheetBoard, "F1", "Status")
	f.SetCellValue(sheetBoard, "G1", "Tenggat Waktu")
	f.SetCellValue(sheetBoard, "H1", "Cover Tugas")
	f.SetCellValue(sheetBoard, "I1", "Berkas")
	f.SetCellValue(sheetBoard, "J1", "Checklist dan Item")
	f.SetCellValue(sheetBoard, "K1", "Histori dan Komentar")
	f.SetCellValue(sheetBoard, "L1", "Dibuat Oleh")
	f.SetCellValue(sheetBoard, "M1", "Tanggal Dibuat")

	for i, v := range dataTask {
		fileData, err := s.TaskFileRepository.FindByTaskId(ctx, v.ID, true)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		commentData, err := s.TaskCommentRepository.FindByTaskId(ctx, v.ID, true)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		checklistData, err := s.TaskChecklistRepository.FindByTaskId(ctx, v.ID, true)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		colA := fmt.Sprintf("A%d", i+2)
		colB := fmt.Sprintf("B%d", i+2)
		colC := fmt.Sprintf("C%d", i+2)
		colD := fmt.Sprintf("D%d", i+2)
		colE := fmt.Sprintf("E%d", i+2)
		colF := fmt.Sprintf("F%d", i+2)
		colG := fmt.Sprintf("G%d", i+2)
		colH := fmt.Sprintf("H%d", i+2)
		colI := fmt.Sprintf("I%d", i+2)
		colJ := fmt.Sprintf("J%d", i+2)
		colK := fmt.Sprintf("K%d", i+2)
		colL := fmt.Sprintf("L%d", i+2)
		colM := fmt.Sprintf("M%d", i+2)
		no := i + 1
		f.SetCellValue(sheetBoard, colA, no)
		f.SetCellValue(sheetBoard, colB, v.Title)
		if v.Description != nil {
			f.SetCellValue(sheetBoard, colC, *v.Description)
		} else {
			f.SetCellValue(sheetBoard, colC, "-")
		}
		if v.AssignToUser != nil {
			var assignToUser []map[string]interface{}
			assignToUserArr := general.StringToArrayInt(v.AssignToUser)
			for _, u := range assignToUserArr {
				dataUser, err := s.UserRepository.FindById(ctx, u)
				if err != nil && err.Error() != "record not found" {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
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
			var valAssignToUser []string
			for _, j := range assignToUser {
				name, nameOk := j["name"].(string)
				role, roleOk := j["role"].(string)
				divisi, divisiOk := j["divisi"].(string)
				if nameOk && roleOk && divisiOk {
					valAssignToUser = append(valAssignToUser, fmt.Sprintf("%s (%s - %s)", name, role, divisi))
				}
			}
			f.SetCellValue(sheetBoard, colD, strings.Join(valAssignToUser, "\n"))
		} else {
			f.SetCellValue(sheetBoard, colD, "-")
		}
		if v.Label != nil {
			var label []map[string]interface{}
			labelArr := general.StringToArrayInt(v.Label)
			for _, v := range labelArr {
				dataLabel, err := s.TaskLabelRepository.FindById(ctx, v)
				if err != nil && err.Error() != "record not found" {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
				if dataLabel != nil {
					label = append(label, map[string]interface{}{
						"id":    dataLabel.ID,
						"title": dataLabel.Title,
						"color": dataLabel.Color,
					})
				}
			}
			var valLabel []string
			for _, j := range label {
				title, titleOk := j["title"].(string)
				color, colorOk := j["color"].(string)
				if titleOk && colorOk {
					valLabel = append(valLabel, fmt.Sprintf("%s (%s)", title, general.GetColorNameFromCode(color)))
				}
			}
			f.SetCellValue(sheetBoard, colE, strings.Join(valLabel, "\n"))
		} else {
			f.SetCellValue(sheetBoard, colE, "-")
		}
		if v.IsCompleted {
			f.SetCellValue(sheetBoard, colF, "Selesai")
		} else {
			f.SetCellValue(sheetBoard, colF, "Belum Selesai")
		}
		if v.DueDate != nil {
			f.SetCellValue(sheetBoard, colG, v.DueDate.Format("2006-01-02 15:04:05"))
		} else {
			f.SetCellValue(sheetBoard, colG, "-")
		}
		if v.Cover != nil {
			cover, err := gdrive.GetFile(s.sDrive, *v.Cover)
			if err != nil {
				return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "cover not found")
			}
			f.SetCellValue(sheetBoard, colH, cover.WebContentLink)
			f.SetCellHyperLink(sheetBoard, colH, general.IsValidURL(cover.WebContentLink), "External")
		} else {
			f.SetCellValue(sheetBoard, colH, "-")
		}

		var resFile []map[string]interface{}
		for _, j := range fileData {
			fileDrive, err := gdrive.GetFile(s.sDrive, j.File)
			if err != nil {
				return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "file not found")
			}
			resFile = append(resFile, map[string]interface{}{
				"file": map[string]interface{}{
					"content": fileDrive.WebContentLink,
				},
				"file_name": j.FileName,
			})
		}
		if resFile != nil {
			var valFile []string
			for _, j := range resFile {
				fileNameRaw := j["file_name"]
				fileRaw := j["file"]

				fileName, okName := fileNameRaw.(string)
				fileMap, okFile := fileRaw.(map[string]interface{})

				if okName && okFile {
					viewSavedRaw := fileMap["content"]
					viewSaved, okView := viewSavedRaw.(string)

					if okView {
						valFile = append(valFile, fmt.Sprintf("%s (%s)", fileName, viewSaved))
					}
				}
			}
			f.SetCellValue(sheetBoard, colI, strings.Join(valFile, "\n"))
		} else {
			f.SetCellValue(sheetBoard, colI, "-")
		}

		var resChecklist []map[string]interface{}
		for _, v := range checklistData {
			dataChecklistItem, err := s.ChecklistItemRepository.FindByTaskChecklistIdArr(ctx, v.ID, true)
			if err != nil && err.Error() != "record not found" {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}

			var dataChecklistItemArr []map[string]interface{}
			isChecklistItemCompleted := 0
			for _, ci := range dataChecklistItem {
				if ci.IsCompleted {
					isChecklistItemCompleted++
				}
				checklistItem := map[string]interface{}{
					"title":          ci.Title,
					"assign_to_user": ci.AssignToUser,
					"due_date":       ci.DueDate,
					"is_completed":   ci.IsCompleted,
				}

				if ci.AssignToUser != nil {
					var assignToUser []map[string]interface{}
					assignToUserArr := general.StringToArrayInt(ci.AssignToUser)
					for _, v := range assignToUserArr {
						dataUser, err := s.UserRepository.FindById(ctx, v)
						if err != nil && err.Error() != "record not found" {
							return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
						}
						if dataUser != nil {
							assignToUser = append(assignToUser, map[string]interface{}{
								"name":   dataUser.Name,
								"role":   dataUser.Role.Name,
								"divisi": dataUser.Divisi.Name,
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
				"title":            v.Title,
				"check_persentase": strconv.Itoa(checkPersentase) + "%",
				"item": map[string]interface{}{
					"count": len(dataChecklistItem),
					"data":  dataChecklistItemArr,
				},
			})
		}
		if resChecklist != nil {
			var valChecklist []string
			for _, j := range resChecklist {
				itemRaw := j["item"]
				valItemChecklist := []string{}

				itemMap, okItem := itemRaw.(map[string]interface{})
				if !okItem {
					continue
				}

				dataRaw, okData := itemMap["data"]
				if !okData {
					continue
				}

				dataSliceInterface, okSlice := dataRaw.([]map[string]interface{})
				if !okSlice {
					continue
				}

				for i, k := range dataSliceInterface {
					no := i + 1
					titleRaw := k["title"]
					isCompletedRaw := k["is_completed"]
					status := "Belum Selesai"
					if isCompleted, ok := isCompletedRaw.(bool); ok && isCompleted {
						status = "Selesai"
					}

					assignToUserStr := ""
					if ar, ok := k["assign_to_user"].(map[string]interface{}); ok && ar != nil {
						if assignMap, ok := k["assign_to_user"].(map[string]interface{}); ok {
							if ux, ok := assignMap["data"].([]map[string]interface{}); ok {
								var names []string
								for _, u := range ux {
									if n, ok := u["name"].(string); ok {
										if r, ok := u["role"].(string); ok {
											if d, ok := u["divisi"].(string); ok {
												names = append(names, fmt.Sprintf("%s (%s %s)", n, r, d))
											}
										}
									}
								}
								assignToUserStr = general.FormatNamesFromArray(names)
							}
						}
					}

					dueDateStr := ""
					if dr, ok := k["due_date"].(*time.Time); ok && dr != nil {
						dueDateStr = dr.Format("2006-01-02 15:04:05")
					}

					titleStr, _ := titleRaw.(string)
					if assignToUserStr == "" {
						assignToUserStr = "no member"
					}
					if dueDateStr == "" {
						dueDateStr = "no due date"
					}
					valItemChecklist = append(valItemChecklist,
						fmt.Sprintf("%d. %s - %s - Member: %s - Due date: %s",
							no, titleStr, status, assignToUserStr, dueDateStr,
						),
					)
				}

				title, _ := j["title"].(string)
				persentase, _ := j["check_persentase"].(string)
				valChecklist = append(valChecklist,
					fmt.Sprintf("%s (%s): \n%s",
						title, persentase,
						strings.Join(valItemChecklist, "\n"),
					),
				)
			}
			f.SetCellValue(sheetBoard, colJ, strings.Join(valChecklist, "\n"))
		} else {
			f.SetCellValue(sheetBoard, colJ, "-")
		}

		var resComment []map[string]interface{}
		for _, v := range commentData {
			resComment = append(resComment, map[string]interface{}{
				"id":         v.ID,
				"task_id":    v.TaskId,
				"comment":    v.Comment,
				"is_delete":  v.IsDelete,
				"is_history": v.IsHistory,
				"created_at": v.CreatedAt.Format("2006-01-02 15:04:05"),
				"created_by": map[string]interface{}{
					"name":   v.CreateBy.Name,
					"role":   v.CreateBy.Role.Name,
					"divisi": v.CreateBy.Divisi.Name,
				},
			})
		}
		if resComment != nil {
			var valComment []string
			for _, j := range resComment {
				createdBy := j["created_by"].(map[string]interface{})
				name := createdBy["name"].(string)
				role := createdBy["role"].(string)
				divisi := createdBy["divisi"].(string)
				comment := j["comment"].(string)
				createdAt := j["created_at"].(string)

				history := ""
				if isHistory, ok := j["is_history"].(bool); ok && isHistory {
					history = "(history)"
				}

				if name == "System" {
					valComment = append(valComment, fmt.Sprintf("%s <%s> - %s %s", name, createdAt, comment, history))
				} else {
					valComment = append(valComment, fmt.Sprintf("%s (%s - %s) <%s> - %s %s", name, role, divisi, createdAt, comment, history))
				}
			}
			f.SetCellValue(sheetBoard, colK, strings.Join(valComment, "\n"))
		} else {
			f.SetCellValue(sheetBoard, colK, "-")
		}

		f.SetCellValue(sheetBoard, colL, fmt.Sprintf("%s (%s - %s)", v.CreateBy.Name, v.CreateBy.Role.Name, v.CreateBy.Divisi.Name))
		f.SetCellValue(sheetBoard, colM, v.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}
