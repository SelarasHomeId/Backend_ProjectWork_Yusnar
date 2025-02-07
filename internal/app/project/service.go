package project

import (
	"errors"
	"net/http"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/dto"
	"selarashomeid/internal/factory"
	"selarashomeid/internal/model"
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/constant"
	"selarashomeid/pkg/util/response"
	"selarashomeid/pkg/util/trxmanager"

	"gorm.io/gorm"
)

type Service interface {
	Create(ctx *abstraction.Context, payload *dto.ProjectCreateRequest) (map[string]interface{}, error)
	Find(ctx *abstraction.Context) (map[string]interface{}, error)
	Update(ctx *abstraction.Context, payload *dto.ProjectUpdateRequest) (map[string]interface{}, error)
	Delete(ctx *abstraction.Context, payload *dto.ProjectDeleteByIDRequest) (map[string]interface{}, error)
}

type service struct {
	ProjectRepository   repository.Project
	UserRepository      repository.User
	WorkspaceRepository repository.Workspace
	BoardRepository     repository.Board

	DB *gorm.DB
}

func NewService(f *factory.Factory) Service {
	return &service{
		ProjectRepository:   f.ProjectRepository,
		UserRepository:      f.UserRepository,
		WorkspaceRepository: f.WorkspaceRepository,
		BoardRepository:     f.BoardRepository,

		DB: f.Db,
	}
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.ProjectCreateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		if ctx.Auth.RoleID != constant.ROLE_ID_ADMIN {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "this role is not permitted")
		}

		dataAllProject, err := s.ProjectRepository.Find(ctx)
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
			modelProject.Location = *payload.Location
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

		return nil
	}); err != nil {
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
	data, err := s.ProjectRepository.Find(ctx)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	count, err := s.ProjectRepository.Count(ctx)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	for _, v := range data {
		res = append(res, map[string]interface{}{
			"id":         v.ID,
			"name":       v.Name,
			"location":   v.Location,
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

func (s *service) Update(ctx *abstraction.Context, payload *dto.ProjectUpdateRequest) (map[string]interface{}, error) {
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
			newProjectData.Location = *payload.Location
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
		return nil
	}); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message": "success delete!",
	}, nil
}
