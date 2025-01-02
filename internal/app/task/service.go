package task

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
	"selarashomeid/pkg/gdrive"
	"selarashomeid/pkg/util/general"
	"selarashomeid/pkg/util/response"
	"selarashomeid/pkg/util/trxmanager"

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
}

type service struct {
	TaskRepository        repository.Task
	BoardRepository       repository.Board
	WorkspaceRepository   repository.Workspace
	UserRepository        repository.User
	TaskFileRepository    repository.TaskFile
	TaskCommentRepository repository.TaskComment
	NotifikasiRepository  repository.Notifikasi

	DB     *gorm.DB
	sDrive *drive.Service
	fDrive *drive.File
}

func NewService(f *factory.Factory) Service {
	return &service{
		TaskRepository:        f.TaskRepository,
		BoardRepository:       f.BoardRepository,
		WorkspaceRepository:   f.WorkspaceRepository,
		UserRepository:        f.UserRepository,
		TaskFileRepository:    f.TaskFileRepository,
		TaskCommentRepository: f.TaskCommentRepository,
		NotifikasiRepository:  f.NotifikasiRepository,

		DB:     f.Db,
		sDrive: f.GDrive.Service,
		fDrive: f.GDrive.Folder,
	}
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.TaskCreateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		userLogin, err := s.UserRepository.FindById(ctx, ctx.Auth.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		userAdmin, err := s.UserRepository.FindByRoleId(ctx, constant.ROLE_ID_ADMIN)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		boardData, err := s.BoardRepository.FindById(ctx, *payload.BoardId)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if boardData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "board not found")
		}

		modelTask := &model.TaskEntityModel{
			Context: ctx,
			TaskEntity: model.TaskEntity{
				BoardId:     *payload.BoardId,
				Title:       *payload.Title,
				IsCompleted: false,
				IsDelete:    false,
			},
		}
		if err := s.TaskRepository.Create(ctx, modelTask).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		newBoardData := new(model.BoardEntityModel)
		newBoardData.Context = ctx
		newBoardData.ID = *payload.BoardId
		newBoardData.TaskTotal = boardData.TaskTotal + 1
		if err = s.BoardRepository.UpdateTaskTotalById(ctx, newBoardData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		modelNotifikasi := new(model.NotifikasiEntityModel)
		modelNotifikasi.Context = ctx
		modelNotifikasi.Title = fmt.Sprintf("Tugas baru telah dibuat oleh %s", userLogin.Name)
		modelNotifikasi.Message = modelTask.Title
		modelNotifikasi.IsRead = false
		modelNotifikasi.UserId = userAdmin.ID
		if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
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

func (s *service) Delete(ctx *abstraction.Context, payload *dto.TaskDeleteByIDRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		taskData, err := s.TaskRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if taskData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "task not found")
		}

		boardData, err := s.BoardRepository.FindById(ctx, taskData.BoardId)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if boardData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "board not found")
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

		return nil
	}); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message": "success delete!",
	}, nil
}

func (s *service) FindByBoardId(ctx *abstraction.Context, payload *dto.TaskFindByBoardIDRequest) (map[string]interface{}, error) {
	var res []map[string]interface{}

	boardData, err := s.BoardRepository.FindById(ctx, payload.BoardID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	if boardData == nil {
		return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "board not found")
	}

	data, err := s.TaskRepository.FindByBoardIdArr(ctx, payload.BoardID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	count, err := s.TaskRepository.CountByBoardIdArr(ctx, payload.BoardID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}

	for _, v := range data {
		fileData, err := s.TaskFileRepository.FindByTaskId(ctx, v.ID)
		if err != nil && err.Error() != "record not found" {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		commentData, err := s.TaskCommentRepository.FindByTaskId(ctx, v.ID)
		if err != nil && err.Error() != "record not found" {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		description := false
		if v.Description != nil && *v.Description != "" {
			description = true
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
			"file":           len(fileData),
			"comment":        len(commentData),
			"is_delete":      v.IsDelete,
			"created_at":     v.CreatedAt,
			"updated_at":     v.UpdatedAt,
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
		}

		if v.AssignToUser != nil {
			var assignToUser []map[string]interface{}
			assignToUserArr := general.AssignToUserStringToArray(*v.AssignToUser)
			for _, v := range assignToUserArr {
				dataUser, err := s.UserRepository.FindById(ctx, v)
				if err != nil && err.Error() != "record not found" {
					return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
				assignToUser = append(assignToUser, map[string]interface{}{
					"id":    dataUser.ID,
					"name":  dataUser.Name,
					"email": dataUser.Email,
				})
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
				"view":    "https://lh3.googleusercontent.com/d/" + *v.Cover,
				"content": cover.WebContentLink,
				"name":    cover.Name,
			}
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

		newTaskData := new(model.TaskEntityModel)
		newTaskData.Context = ctx
		newTaskData.ID = payload.ID
		if payload.BoardId != nil {
			boardData, err := s.BoardRepository.FindById(ctx, *payload.BoardId)
			if err != nil && err.Error() != "record not found" {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
			if boardData == nil {
				return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "board not found")
			}
			changesTaskTotal.NewBoard = &boardData.ID

			newTaskData.BoardId = *payload.BoardId
		}
		if payload.Title != nil {
			newTaskData.Title = *payload.Title
		}
		if payload.Description != nil {
			newTaskData.Description = payload.Description
		}
		if payload.AssignToUser != nil {
			for _, v := range payload.AssignToUser {
				userData, err := s.UserRepository.FindById(ctx, v)
				if err != nil && err.Error() != "record not found" {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
				if userData == nil {
					return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "user assign to not found")
				}
			}
			strAssignToUser := general.AssignToUserArrayToString(payload.AssignToUser)
			if strAssignToUser == "" {
				newTaskData.AssignToUser = nil
			} else {
				newTaskData.AssignToUser = &strAssignToUser
			}
		}
		if payload.Label != nil {
			newTaskData.Label = payload.Label
		}
		if payload.IsCompleted != nil {
			newTaskData.IsCompleted = *payload.IsCompleted
			if err = s.TaskRepository.UpdateCompleted(ctx, newTaskData).Error; err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
		}
		if payload.DueDate != nil {
			parsedDueDate, err := general.Parse("2006-01-02", *payload.DueDate)
			if err != nil {
				return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "err parse due date:"+err.Error())
			}
			newTaskData.DueDate = &parsedDueDate
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

			if taskData.Cover != nil {
				err = gdrive.DeleteFile(s.sDrive, *taskData.Cover)
				if err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
		}
		if err = s.TaskRepository.Update(ctx, newTaskData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		if changesTaskTotal.NewBoard != nil {
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
	fileData, err := s.TaskFileRepository.FindByTaskId(ctx, payload.ID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	commentData, err := s.TaskCommentRepository.FindByTaskId(ctx, payload.ID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	if data != nil {
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
			"file":           nil,
			"comment":        nil,
			"is_delete":      data.IsDelete,
			"created_at":     data.CreatedAt,
			"updated_at":     data.UpdatedAt,
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
		}

		if data.AssignToUser != nil {
			var assignToUser []map[string]interface{}
			assignToUserArr := general.AssignToUserStringToArray(*data.AssignToUser)
			for _, v := range assignToUserArr {
				dataUser, err := s.UserRepository.FindById(ctx, v)
				if err != nil && err.Error() != "record not found" {
					return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
				assignToUser = append(assignToUser, map[string]interface{}{
					"id":    dataUser.ID,
					"name":  dataUser.Name,
					"email": dataUser.Email,
				})
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
				"view":    "https://lh3.googleusercontent.com/d/" + *data.Cover,
				"content": cover.WebContentLink,
				"name":    cover.Name,
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
					"view":    "https://lh3.googleusercontent.com/d/" + v.File,
					"content": fileDrive.WebContentLink,
					"name":    fileDrive.Name,
				},
				"is_delete":  v.IsDelete,
				"created_at": v.CreatedAt,
				"updated_at": v.UpdatedAt,
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
				"created_at": v.CreatedAt,
				"updated_at": v.UpdatedAt,
			})
		}
		res["comment"] = map[string]interface{}{
			"count": len(commentData),
			"data":  resComment,
		}

	}

	return map[string]interface{}{
		"data": res,
	}, nil
}
