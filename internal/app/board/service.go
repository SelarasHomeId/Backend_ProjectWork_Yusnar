package board

import (
	"errors"
	"net/http"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/dto"
	"selarashomeid/internal/factory"
	"selarashomeid/internal/model"
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/util/response"
	"selarashomeid/pkg/util/trxmanager"

	"gorm.io/gorm"
)

type Service interface {
	Create(ctx *abstraction.Context, payload *dto.BoardCreateRequest) (map[string]interface{}, error)
	Delete(ctx *abstraction.Context, payload *dto.BoardDeleteByIDRequest) (map[string]interface{}, error)
	Update(ctx *abstraction.Context, payload *dto.BoardUpdateRequest) (map[string]interface{}, error)
	FindByWorkspaceId(ctx *abstraction.Context, payload *dto.BoardFindByWorkspaceIDRequest) (map[string]interface{}, error)
}

type service struct {
	BoardRepository     repository.Board
	WorkspaceRepository repository.Workspace

	DB *gorm.DB
}

func NewService(f *factory.Factory) Service {
	return &service{
		BoardRepository:     f.BoardRepository,
		WorkspaceRepository: f.WorkspaceRepository,

		DB: f.Db,
	}
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.BoardCreateRequest) (map[string]interface{}, error) {
	boardData := new(model.BoardEntityModel)
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		workspaceData, err := s.WorkspaceRepository.FindById(ctx, *payload.WorkspaceId)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if workspaceData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "workspace not found")
		}

		dataBoardInWorkspace, err := s.BoardRepository.FindByWorkspaceId(ctx, *payload.WorkspaceId)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		sortNum := 1
		if dataBoardInWorkspace != nil {
			sortNum = dataBoardInWorkspace.SortNumber + 1
		}

		modelBoard := &model.BoardEntityModel{
			Context: ctx,
			BoardEntity: model.BoardEntity{
				WorkspaceId: *payload.WorkspaceId,
				Name:        *payload.Name,
				TaskTotal:   0,
				SortNumber:  sortNum,
				IsDelete:    false,
			},
		}
		if err := s.BoardRepository.Create(ctx, modelBoard).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		boardData = modelBoard
		return nil
	}); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message":      "success create!",
		"id":           boardData.ID,
		"name":         boardData.Name,
		"sort_number":  boardData.SortNumber,
		"task_total":   boardData.TaskTotal,
		"workspace_id": boardData.WorkspaceId,
	}, nil
}

func (s *service) Delete(ctx *abstraction.Context, payload *dto.BoardDeleteByIDRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		boardData, err := s.BoardRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if boardData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "board not found")
		}

		newBoardData := new(model.BoardEntityModel)
		newBoardData.Context = ctx
		newBoardData.ID = boardData.ID
		newBoardData.IsDelete = true

		if err = s.BoardRepository.Update(ctx, newBoardData).Error; err != nil {
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

func (s *service) Update(ctx *abstraction.Context, payload *dto.BoardUpdateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		boardData, err := s.BoardRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if boardData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "board not found")
		}

		newBoardData := new(model.BoardEntityModel)
		newBoardData.Context = ctx
		newBoardData.ID = payload.ID
		if payload.Name != nil {
			newBoardData.Name = *payload.Name
		}
		if payload.WorkspaceId != nil {
			workspaceData, err := s.WorkspaceRepository.FindById(ctx, *payload.WorkspaceId)
			if err != nil && err.Error() != "record not found" {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
			if workspaceData == nil {
				return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "workspace not found")
			}

			newBoardData.WorkspaceId = *payload.WorkspaceId
		}
		if payload.SortNumber != nil {
			newBoardData.SortNumber = *payload.SortNumber
		}

		if err = s.BoardRepository.Update(ctx, newBoardData).Error; err != nil {
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

func (s *service) FindByWorkspaceId(ctx *abstraction.Context, payload *dto.BoardFindByWorkspaceIDRequest) (map[string]interface{}, error) {
	var res []map[string]interface{} = nil

	workspaceData, err := s.WorkspaceRepository.FindById(ctx, payload.WorkspaceID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	if workspaceData == nil {
		return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "workspace not found")
	}

	data, err := s.BoardRepository.FindByWorkspaceIdArr(ctx, payload.WorkspaceID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	count, err := s.BoardRepository.CountByWorkspaceIdArr(ctx, payload.WorkspaceID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}

	for _, v := range data {
		res = append(res, map[string]interface{}{
			"id":           v.ID,
			"workspace_id": v.WorkspaceId,
			"name":         v.Name,
			"task_total":   v.TaskTotal,
			"sort_number":  v.SortNumber,
			"is_delete":    v.IsDelete,
			"created_at":   v.CreatedAt,
			"updated_at":   v.UpdatedAt,
		})
	}

	return map[string]interface{}{
		"count": count,
		"data":  res,
	}, nil
}
