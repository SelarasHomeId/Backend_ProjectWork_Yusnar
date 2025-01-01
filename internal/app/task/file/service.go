package file

import (
	"errors"
	"fmt"
	"net/http"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/dto"
	"selarashomeid/internal/factory"
	"selarashomeid/internal/model"
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/gdrive"
	"selarashomeid/pkg/util/general"
	"selarashomeid/pkg/util/response"
	"selarashomeid/pkg/util/trxmanager"

	"github.com/sirupsen/logrus"
	"google.golang.org/api/drive/v3"
	"gorm.io/gorm"
)

type Service interface {
	Create(ctx *abstraction.Context, payload *dto.TaskFileCreateRequest) (map[string]interface{}, error)
	FindByTaskId(ctx *abstraction.Context, payload *dto.TaskFileFindByTaskIDRequest) (map[string]interface{}, error)
	Delete(ctx *abstraction.Context, payload *dto.TaskFileDeleteByIDRequest) (map[string]interface{}, error)
	Update(ctx *abstraction.Context, payload *dto.TaskFileUpdateRequest) (map[string]interface{}, error)
	FindById(ctx *abstraction.Context, payload *dto.TaskFileFindByIDRequest) (map[string]interface{}, error)
}

type service struct {
	TaskRepository     repository.Task
	TaskFileRepository repository.TaskFile

	DB     *gorm.DB
	sDrive *drive.Service
	fDrive *drive.File
}

func NewService(f *factory.Factory) Service {
	return &service{
		TaskRepository:     f.TaskRepository,
		TaskFileRepository: f.TaskFileRepository,

		DB:     f.Db,
		sDrive: f.GDrive.Service,
		fDrive: f.GDrive.Folder,
	}
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.TaskFileCreateRequest) (map[string]interface{}, error) {
	var allFileUploaded []string = nil
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		taskData, err := s.TaskRepository.FindById(ctx, *payload.TaskId)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if taskData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "task not found")
		}

		for _, file := range payload.File {
			f, err := file.Open()
			if err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
			defer f.Close()

			isFileAvailable, fullFileName := general.ValidateFileUpload(file.Filename)
			if !isFileAvailable {
				return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), fmt.Sprintf("file format for %s is not approved", file.Filename))
			}

			newFile, err := gdrive.CreateFile(s.sDrive, fullFileName, "application/octet-stream", f, s.fDrive.Id)
			if err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
			allFileUploaded = append(allFileUploaded, newFile.Id)

			modelTaskFile := &model.TaskFileEntityModel{
				Context: ctx,
				TaskFileEntity: model.TaskFileEntity{
					TaskId:   *payload.TaskId,
					File:     newFile.Id,
					IsDelete: false,
				},
			}

			if err := s.TaskFileRepository.Create(ctx, modelTaskFile).Error; err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
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
		for _, v := range allFileUploaded {
			errDel := gdrive.DeleteFile(s.sDrive, v)
			if errDel != nil {
				logrus.Error("error delete file for error trxmanager:", errDel.Error())
			}
		}
		return nil, err
	}
	return map[string]interface{}{
		"message": "success create!",
	}, nil
}

func (s *service) FindByTaskId(ctx *abstraction.Context, payload *dto.TaskFileFindByTaskIDRequest) (map[string]interface{}, error) {
	var res []map[string]interface{}

	taskData, err := s.TaskRepository.FindById(ctx, payload.TaskId)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	if taskData == nil {
		return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "task not found")
	}

	data, err := s.TaskFileRepository.FindByTaskId(ctx, payload.TaskId)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	count, err := s.TaskFileRepository.CountByTaskId(ctx, payload.TaskId)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}

	for _, v := range data {
		file, err := gdrive.GetFile(s.sDrive, v.File)
		if err != nil {
			return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "file not found")
		}

		res = append(res, map[string]interface{}{
			"id":      v.ID,
			"task_id": v.TaskId,
			"file": map[string]interface{}{
				"view":    "https://lh3.googleusercontent.com/d/" + v.File,
				"content": file.WebContentLink,
				"name":    file.Name,
			},
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

func (s *service) Delete(ctx *abstraction.Context, payload *dto.TaskFileDeleteByIDRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		taskFileData, err := s.TaskFileRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if taskFileData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "task file not found")
		}

		newTaskFileData := new(model.TaskFileEntityModel)
		newTaskFileData.Context = ctx
		newTaskFileData.ID = payload.ID
		newTaskFileData.IsDelete = true
		if err = s.TaskFileRepository.Update(ctx, newTaskFileData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		newTaskData := new(model.TaskEntityModel)
		newTaskData.Context = ctx
		newTaskData.ID = taskFileData.TaskId
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

func (s *service) Update(ctx *abstraction.Context, payload *dto.TaskFileUpdateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		taskFileData, err := s.TaskFileRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if taskFileData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "task file not found")
		}

		if payload.Name != nil {
			_, err := gdrive.RenameFile(s.sDrive, taskFileData.File, *payload.Name)
			if err != nil {
				return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "failed rename file: "+err.Error())
			}
		}

		newTaskFileData := new(model.TaskFileEntityModel)
		newTaskFileData.Context = ctx
		newTaskFileData.ID = payload.ID
		newTaskFileData.UpdatedAt = general.NowLocal()
		if err = s.TaskFileRepository.Update(ctx, newTaskFileData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		newTaskData := new(model.TaskEntityModel)
		newTaskData.Context = ctx
		newTaskData.ID = taskFileData.TaskId
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

func (s *service) FindById(ctx *abstraction.Context, payload *dto.TaskFileFindByIDRequest) (map[string]interface{}, error) {
	var res map[string]interface{} = nil

	data, err := s.TaskFileRepository.FindById(ctx, payload.ID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	if data != nil {
		file, err := gdrive.GetFile(s.sDrive, data.File)
		if err != nil {
			return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "file not found")
		}
		res = map[string]interface{}{
			"id":      data.ID,
			"task_id": data.TaskId,
			"file": map[string]interface{}{
				"view":    "https://lh3.googleusercontent.com/d/" + data.File,
				"content": file.WebContentLink,
				"name":    file.Name,
			},
			"is_delete":  data.IsDelete,
			"created_at": data.CreatedAt,
			"updated_at": data.UpdatedAt,
		}
	}

	return map[string]interface{}{
		"data": res,
	}, nil
}
