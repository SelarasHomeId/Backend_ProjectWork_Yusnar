package project

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
	Create(ctx *abstraction.Context, payload *dto.ProjectCreateRequest) (map[string]interface{}, error)
	Find(ctx *abstraction.Context) (map[string]interface{}, error)
	Update(ctx *abstraction.Context, payload *dto.ProjectUpdateRequest) (map[string]interface{}, error)
	Delete(ctx *abstraction.Context, payload *dto.ProjectDeleteByIDRequest) (map[string]interface{}, error)
	FindById(ctx *abstraction.Context, payload *dto.ProjectFindByIDRequest) (map[string]interface{}, error)
}

type service struct {
	ProjectRepository   repository.Project
	UserRepository      repository.User
	WorkspaceRepository repository.Workspace
	BoardRepository     repository.Board
	TaskRepository      repository.Task

	DB     *gorm.DB
	sDrive *drive.Service
	fDrive *drive.File
}

func NewService(f *factory.Factory) Service {
	return &service{
		ProjectRepository:   f.ProjectRepository,
		UserRepository:      f.UserRepository,
		WorkspaceRepository: f.WorkspaceRepository,
		BoardRepository:     f.BoardRepository,
		TaskRepository:      f.TaskRepository,

		DB:     f.Db,
		sDrive: f.GDrive.Service,
		fDrive: f.GDrive.FolderProjectCover,
	}
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.ProjectCreateRequest) (map[string]interface{}, error) {
	var allFileUploaded []string = nil
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		if ctx.Auth.RoleID != constant.ROLE_ID_ADMIN {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "this role is not permitted")
		}

		dataAllProject, err := s.ProjectRepository.Find(ctx, true)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		for _, v := range dataAllProject {
			if payload.Name == v.Name {
				return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "project already exist")
			}
		}

		modelProject := &model.ProjectEntityModel{
			Context: ctx,
			ProjectEntity: model.ProjectEntity{
				Name:     payload.Name,
				IsDelete: false,
			},
		}

		if payload.Location != nil {
			modelProject.Location = payload.Location
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

			modelProject.Cover = &newFile.Id
			modelProject.CoverName = &newFile.Name
		}

		if err := s.ProjectRepository.Create(ctx, modelProject).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		modelWorkspace := &model.WorkspaceEntityModel{
			Context: ctx,
			WorkspaceEntity: model.WorkspaceEntity{
				ProjectId: modelProject.ID,
				Name:      modelProject.Name,
				IsDelete:  false,
			},
		}
		if err := s.WorkspaceRepository.Create(ctx, modelWorkspace).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		modelBoard := &model.BoardEntityModel{
			Context: ctx,
			BoardEntity: model.BoardEntity{
				WorkspaceId: modelWorkspace.ID,
				Name:        "To Do",
				TaskTotal:   0,
				SortNumber:  1,
				IsDelete:    false,
			},
		}
		if err := s.BoardRepository.Create(ctx, modelBoard).Error; err != nil {
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

func (s *service) Find(ctx *abstraction.Context) (map[string]interface{}, error) {
	var res []map[string]interface{} = nil
	if ctx.Auth.RoleID != constant.ROLE_ID_ADMIN {
		return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "this role is not permitted")
	}
	data, err := s.ProjectRepository.Find(ctx, false)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	count, err := s.ProjectRepository.Count(ctx)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	for _, v := range data {
		project := map[string]interface{}{
			"id":         v.ID,
			"name":       v.Name,
			"location":   v.Location,
			"cover":      v.Cover,
			"cover_name": v.CoverName,
			"is_delete":  v.IsDelete,
			"created_at": general.FormatWithZWithoutChangingTime(v.CreatedAt),
			"updated_at": general.FormatWithZWithoutChangingTime(*v.UpdatedAt),
		}

		if v.Cover != nil {
			cover, err := gdrive.GetFile(s.sDrive, *v.Cover)
			if err != nil {
				return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "cover not found")
			}
			project["cover"] = map[string]interface{}{
				"view":       "https://lh3.googleusercontent.com/d/" + *v.Cover,
				"view_saved": general.ConvertLinkToFileSaved(cover.WebContentLink),
				"content":    cover.WebContentLink,
				"ext":        cover.FileExtension,
				"name":       cover.Name,
				"id":         cover.Id,
			}
		}

		res = append(res, project)
	}
	return map[string]interface{}{
		"count": count,
		"data":  res,
	}, nil
}

func (s *service) Update(ctx *abstraction.Context, payload *dto.ProjectUpdateRequest) (map[string]interface{}, error) {
	var allFileUploaded []string = nil
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		if ctx.Auth.RoleID != constant.ROLE_ID_ADMIN {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "this role is not permitted")
		}

		projectData, err := s.ProjectRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if projectData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "project not found")
		}

		newProjectData := new(model.ProjectEntityModel)
		newProjectData.Context = ctx
		newProjectData.ID = payload.ID
		if payload.Name != nil {
			newProjectData.Name = *payload.Name
		}
		if payload.Location != nil {
			newProjectData.Location = payload.Location
			if *payload.Location == "" {
				if err = s.ProjectRepository.UpdateToNull(ctx, newProjectData, "location").Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
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

			newProjectData.Cover = &newFile.Id
			newProjectData.CoverName = &newFile.Name

			if projectData.Cover != nil {
				err = gdrive.DeleteFile(s.sDrive, *projectData.Cover)
				if err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
		} else {
			if payload.DeleteCover != nil && *payload.DeleteCover {
				errDel := gdrive.DeleteFile(s.sDrive, *projectData.Cover)
				if errDel != nil {
					logrus.Error("error delete file for cover:", errDel.Error())
				}
				if err = s.ProjectRepository.UpdateToNull(ctx, newProjectData, "cover").Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
				if err = s.ProjectRepository.UpdateToNull(ctx, newProjectData, "cover_name").Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
		}

		if err = s.ProjectRepository.Update(ctx, newProjectData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		workspaceData, err := s.WorkspaceRepository.FindByProjectId(ctx, newProjectData.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if workspaceData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "workspace not found")
		}

		newWorkspaceData := new(model.WorkspaceEntityModel)
		newWorkspaceData.Context = ctx
		newWorkspaceData.ProjectId = newProjectData.ID
		if payload.Name != nil {
			newWorkspaceData.Name = *payload.Name
		}

		if err = s.WorkspaceRepository.UpdateByProjectId(ctx, newWorkspaceData).Error; err != nil {
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
		"message": "success update!",
	}, nil
}

func (s *service) Delete(ctx *abstraction.Context, payload *dto.ProjectDeleteByIDRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		if ctx.Auth.RoleID != constant.ROLE_ID_ADMIN {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "this role is not permitted")
		}

		projectData, err := s.ProjectRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if projectData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "project not found")
		}

		newProjectData := new(model.ProjectEntityModel)
		newProjectData.Context = ctx
		newProjectData.ID = projectData.ID
		newProjectData.IsDelete = true

		if err = s.ProjectRepository.Update(ctx, newProjectData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		workspaceData, err := s.WorkspaceRepository.FindByProjectId(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if workspaceData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "workspace not found")
		}

		newWorkspaceData := new(model.WorkspaceEntityModel)
		newWorkspaceData.Context = ctx
		newWorkspaceData.ProjectId = workspaceData.ProjectId
		newWorkspaceData.IsDelete = true

		if err = s.WorkspaceRepository.UpdateByProjectId(ctx, newWorkspaceData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		boardInWorkspace, err := s.BoardRepository.FindByWorkspaceIdArr(ctx, workspaceData.ID, true)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		for _, v := range boardInWorkspace {
			newBoardData := new(model.BoardEntityModel)
			newBoardData.Context = ctx
			newBoardData.ID = v.ID
			newBoardData.IsDelete = true
			newBoardData.TaskTotal = 0
			if err = s.BoardRepository.Update(ctx, newBoardData).Error; err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}

			taskInBoard, err := s.TaskRepository.FindByBoardIdArr(ctx, v.ID, true)
			if err != nil && err.Error() != "record not found" {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
			for _, t := range taskInBoard {
				newTaskData := new(model.TaskEntityModel)
				newTaskData.Context = ctx
				newTaskData.ID = t.ID
				newTaskData.IsDelete = true
				if err = s.TaskRepository.Update(ctx, newTaskData).Error; err != nil {
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

func (s *service) FindById(ctx *abstraction.Context, payload *dto.ProjectFindByIDRequest) (map[string]interface{}, error) {
	var res map[string]interface{} = nil
	data, err := s.ProjectRepository.FindById(ctx, payload.ID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	if data != nil {
		res = map[string]interface{}{
			"id":         data.ID,
			"name":       data.Name,
			"location":   data.Location,
			"cover":      data.Cover,
			"cover_name": data.CoverName,
			"is_delete":  data.IsDelete,
			"created_at": general.FormatWithZWithoutChangingTime(data.CreatedAt),
			"updated_at": general.FormatWithZWithoutChangingTime(*data.UpdatedAt),
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
	}
	return map[string]interface{}{
		"data": res,
	}, nil
}
