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
	"selarashomeid/pkg/constant"
	"selarashomeid/pkg/gdrive"
	"selarashomeid/pkg/util/general"
	"selarashomeid/pkg/util/response"
	"selarashomeid/pkg/util/trxmanager"
	"selarashomeid/pkg/ws"

	"github.com/sirupsen/logrus"
	"google.golang.org/api/drive/v3"
	"gorm.io/gorm"
)

type Service interface {
	Create(ctx *abstraction.Context, payload *dto.TaskFileCreateRequest) (map[string]interface{}, error)
	FindByTaskId(ctx *abstraction.Context, payload *dto.TaskFileFindByTaskIDRequest) (map[string]interface{}, error)
	Delete(ctx *abstraction.Context, payload *dto.TaskFileDeleteByIDRequest) (map[string]interface{}, error)
	Update(ctx *abstraction.Context, payload *dto.TaskFileUpdateRequest) (map[string]interface{}, error)
}

type service struct {
	TaskRepository        repository.Task
	TaskFileRepository    repository.TaskFile
	UserRepository        repository.User
	NotifikasiRepository  repository.Notifikasi
	TaskCommentRepository repository.TaskComment

	DB     *gorm.DB
	sDrive *drive.Service
	fDrive *drive.File
}

func NewService(f *factory.Factory) Service {
	return &service{
		TaskRepository:        f.TaskRepository,
		TaskFileRepository:    f.TaskFileRepository,
		UserRepository:        f.UserRepository,
		NotifikasiRepository:  f.NotifikasiRepository,
		TaskCommentRepository: f.TaskCommentRepository,

		DB:     f.Db,
		sDrive: f.GDrive.Service,
		fDrive: f.GDrive.FolderAttachment,
	}
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.TaskFileCreateRequest) (map[string]interface{}, error) {
	var allFileUploaded []string = nil
	var sendNotifTo []int = nil
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

		var allFileName []string = nil
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
			allFileName = append(allFileName, newFile.Name)

			modelTaskFile := &model.TaskFileEntityModel{
				Context: ctx,
				TaskFileEntity: model.TaskFileEntity{
					TaskId:   payload.TaskId,
					File:     newFile.Id,
					FileName: newFile.Name,
					IsDelete: false,
				},
			}

			if err := s.TaskFileRepository.Create(ctx, modelTaskFile).Error; err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
		}

		newTaskData := new(model.TaskEntityModel)
		newTaskData.Context = ctx
		newTaskData.ID = payload.TaskId
		newTaskData.UpdatedAt = general.NowLocal()
		if err = s.TaskRepository.Update(ctx, newTaskData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		for _, v := range userAdmin {
			if v.ID != userLogin.ID {
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("%s melampirkan berkas pada tugas (%s)", userLogin.Name, taskData.Title)
				modelNotifikasi.Message = general.FormatNamesFromArray(allFileName)
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = v.ID
				modelNotifikasi.TaskId = taskData.ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
				sendNotifTo = append(sendNotifTo, v.ID)
			}
		}

		if taskData.CreateBy.RoleId != constant.ROLE_ID_ADMIN && taskData.CreateBy.ID != userLogin.ID {
			modelNotifikasi := new(model.NotifikasiEntityModel)
			modelNotifikasi.Context = ctx
			modelNotifikasi.Title = fmt.Sprintf("%s melampirkan berkas pada tugas yang anda buat", userLogin.Name)
			modelNotifikasi.Message = general.FormatNamesFromArray(allFileName)
			modelNotifikasi.IsRead = false
			modelNotifikasi.UserId = taskData.CreateBy.ID
			modelNotifikasi.TaskId = taskData.ID
			if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
			sendNotifTo = append(sendNotifTo, taskData.CreateBy.ID)
		}

		for _, v := range assignedMember {
			if v.Role.ID != constant.ROLE_ID_ADMIN && v.ID != taskData.CreateBy.ID && v.ID != userLogin.ID {
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("%s melampirkan berkas pada tugas (%s)", userLogin.Name, taskData.Title)
				modelNotifikasi.Message = general.FormatNamesFromArray(allFileName)
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = v.ID
				modelNotifikasi.TaskId = taskData.ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
				sendNotifTo = append(sendNotifTo, v.ID)
			}
		}

		if err := s.TaskCommentRepository.CreateHistory(ctx, taskData.ID, fmt.Sprintf("Berkas: (%s) dilampirkan oleh %s", general.FormatNamesFromArray(allFileName), userLogin.Name)).Error; err != nil {
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

	for _, v := range general.RemoveDuplicateArrayInt(sendNotifTo) {
		if err := ws.PublishNotificationWithoutTransaction(v, s.DB, ctx); err != nil {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
	}

	return map[string]interface{}{
		"message": "success create!",
	}, nil
}

func (s *service) FindByTaskId(ctx *abstraction.Context, payload *dto.TaskFileFindByTaskIDRequest) (map[string]interface{}, error) {
	var res []map[string]interface{} = nil

	taskData, err := s.TaskRepository.FindById(ctx, payload.TaskId)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	if taskData == nil {
		return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "task not found")
	}

	data, err := s.TaskFileRepository.FindByTaskId(ctx, payload.TaskId, false)
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
				"view":       "https://lh3.googleusercontent.com/d/" + v.File,
				"view_saved": general.ConvertLinkToFileSaved(file.WebContentLink, file.Name, file.FileExtension),
				"content":    file.WebContentLink,
				"ext":        file.FileExtension,
				"name":       file.Name,
			},
			"file_name":  v.FileName,
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

func (s *service) Delete(ctx *abstraction.Context, payload *dto.TaskFileDeleteByIDRequest) (map[string]interface{}, error) {
	var sendNotifTo []int = nil
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		taskFileData, err := s.TaskFileRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if taskFileData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "task file not found")
		}

		taskData, err := s.TaskRepository.FindById(ctx, taskFileData.TaskId)
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

		file, err := gdrive.GetFile(s.sDrive, taskFileData.File)
		if err != nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "file not found")
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

		for _, v := range userAdmin {
			if v.ID != userLogin.ID {
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("%s menghapus berkas pada tugas (%s)", userLogin.Name, taskData.Title)
				modelNotifikasi.Message = fmt.Sprintf("Berkas yang dihapus: %s", file.Name)
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = v.ID
				modelNotifikasi.TaskId = taskData.ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
				sendNotifTo = append(sendNotifTo, v.ID)
			}
		}

		if taskData.CreateBy.RoleId != constant.ROLE_ID_ADMIN && taskData.CreateBy.ID != userLogin.ID {
			modelNotifikasi := new(model.NotifikasiEntityModel)
			modelNotifikasi.Context = ctx
			modelNotifikasi.Title = fmt.Sprintf("%s menghaous berkas pada tugas yang anda buat", userLogin.Name)
			modelNotifikasi.Message = fmt.Sprintf("Berkas yang dihapus: %s", file.Name)
			modelNotifikasi.IsRead = false
			modelNotifikasi.UserId = taskData.CreateBy.ID
			modelNotifikasi.TaskId = taskData.ID
			if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
			sendNotifTo = append(sendNotifTo, taskData.CreateBy.ID)
		}

		for _, v := range assignedMember {
			if v.Role.ID != constant.ROLE_ID_ADMIN && v.ID != taskData.CreateBy.ID && v.ID != userLogin.ID {
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("%s menghapus berkas pada tugas (%s)", userLogin.Name, taskData.Title)
				modelNotifikasi.Message = fmt.Sprintf("Berkas yang dihapus: %s", file.Name)
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = v.ID
				modelNotifikasi.TaskId = taskData.ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
				sendNotifTo = append(sendNotifTo, v.ID)
			}
		}

		if err := s.TaskCommentRepository.CreateHistory(ctx, taskFileData.TaskId, fmt.Sprintf("Berkas: (%s) dihapus oleh %s", file.Name, userLogin.Name)).Error; err != nil {
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

func (s *service) Update(ctx *abstraction.Context, payload *dto.TaskFileUpdateRequest) (map[string]interface{}, error) {
	var sendNotifTo []int = nil
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		taskFileData, err := s.TaskFileRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if taskFileData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "task file not found")
		}

		taskData, err := s.TaskRepository.FindById(ctx, taskFileData.TaskId)
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
		newTaskFileData := new(model.TaskFileEntityModel)
		newTaskFileData.Context = ctx
		newTaskFileData.ID = payload.ID
		if payload.Name != nil {
			_, err := gdrive.RenameFile(s.sDrive, taskFileData.File, *payload.Name)
			if err != nil {
				return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "failed rename file: "+err.Error())
			}
			newTaskFileData.FileName = *payload.Name
			if taskFileData.FileName != newTaskFileData.FileName {
				messageNotif = append(messageNotif, fmt.Sprintf("Berkas berganti nama menjadi (%s)", newTaskFileData.FileName))
			}
		}
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

		for _, v := range userAdmin {
			if v.ID != userLogin.ID {
				for _, d := range messageNotif {
					modelNotifikasi := new(model.NotifikasiEntityModel)
					modelNotifikasi.Context = ctx
					modelNotifikasi.Title = fmt.Sprintf("%s telah mengupdate berkas (%s) di %s", userLogin.Name, taskFileData.FileName, taskData.Title)
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
				modelNotifikasi.Title = fmt.Sprintf("Tugas yang anda buat (%s) terdapat berkas (%s) yang telah diupdate oleh %s", taskData.Title, taskFileData.FileName, userLogin.Name)
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
					modelNotifikasi.Title = fmt.Sprintf("%s telah mengupdate berkas (%s) di %s", userLogin.Name, taskFileData.FileName, taskData.Title)
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
