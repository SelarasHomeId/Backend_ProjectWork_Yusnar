package board

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
	"selarashomeid/pkg/util/general"
	"selarashomeid/pkg/util/response"
	"selarashomeid/pkg/util/trxmanager"
	"selarashomeid/pkg/ws"

	"gorm.io/gorm"
)

type Service interface {
	Create(ctx *abstraction.Context, payload *dto.BoardCreateRequest) (map[string]interface{}, error)
	Delete(ctx *abstraction.Context, payload *dto.BoardDeleteByIDRequest) (map[string]interface{}, error)
	Update(ctx *abstraction.Context, payload *dto.BoardUpdateRequest) (map[string]interface{}, error)
	FindByWorkspaceId(ctx *abstraction.Context, payload *dto.BoardFindByWorkspaceIDRequest) (map[string]interface{}, error)
}

type service struct {
	BoardRepository      repository.Board
	WorkspaceRepository  repository.Workspace
	TaskRepository       repository.Task
	UserRepository       repository.User
	NotifikasiRepository repository.Notifikasi

	DB *gorm.DB
}

func NewService(f *factory.Factory) Service {
	return &service{
		BoardRepository:      f.BoardRepository,
		WorkspaceRepository:  f.WorkspaceRepository,
		TaskRepository:       f.TaskRepository,
		UserRepository:       f.UserRepository,
		NotifikasiRepository: f.NotifikasiRepository,

		DB: f.Db,
	}
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.BoardCreateRequest) (map[string]interface{}, error) {
	boardData := new(model.BoardEntityModel)
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		userLogin, err := s.UserRepository.FindById(ctx, ctx.Auth.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		userAdmin, err := s.UserRepository.FindByRoleIdArr(ctx, constant.ROLE_ID_ADMIN, true)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

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

		for _, v := range userAdmin {
			if v.ID != userLogin.ID {
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("%s telah membuat board baru di %s", userLogin.Name, workspaceData.Name)
				modelNotifikasi.Message = fmt.Sprintf("Board: %s", *payload.Name)
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = v.ID
				modelNotifikasi.TaskId = constant.BLANK_TASK_ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}

				if err := ws.PublishNotificationWithoutTransaction(v.ID, s.DB, ctx); err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}
			}
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
		userLogin, err := s.UserRepository.FindById(ctx, ctx.Auth.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		userAdmin, err := s.UserRepository.FindByRoleIdArr(ctx, constant.ROLE_ID_ADMIN, true)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		boardData, err := s.BoardRepository.FindById(ctx, payload.ID)
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

		newBoardData := new(model.BoardEntityModel)
		newBoardData.Context = ctx
		newBoardData.ID = boardData.ID
		newBoardData.IsDelete = true
		newBoardData.TaskTotal = 0

		if err = s.BoardRepository.Update(ctx, newBoardData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		taskInBoard, err := s.TaskRepository.FindByBoardIdArr(ctx, boardData.ID, true)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		for _, v := range taskInBoard {
			newTaskData := new(model.TaskEntityModel)
			newTaskData.Context = ctx
			newTaskData.ID = v.ID
			newTaskData.IsDelete = true
			if err = s.TaskRepository.Update(ctx, newTaskData).Error; err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
		}

		for _, v := range userAdmin {
			if v.ID != userLogin.ID {
				modelNotifikasi := new(model.NotifikasiEntityModel)
				modelNotifikasi.Context = ctx
				modelNotifikasi.Title = fmt.Sprintf("%s telah menghapus board dari %s", userLogin.Name, workspaceData.Name)
				modelNotifikasi.Message = fmt.Sprintf("Board: %s", boardData.Name)
				modelNotifikasi.IsRead = false
				modelNotifikasi.UserId = v.ID
				modelNotifikasi.TaskId = constant.BLANK_TASK_ID
				if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
					return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
				}

				if err := ws.PublishNotificationWithoutTransaction(v.ID, s.DB, ctx); err != nil {
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

func (s *service) Update(ctx *abstraction.Context, payload *dto.BoardUpdateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		userLogin, err := s.UserRepository.FindById(ctx, ctx.Auth.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		userAdmin, err := s.UserRepository.FindByRoleIdArr(ctx, constant.ROLE_ID_ADMIN, true)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		boardData, err := s.BoardRepository.FindById(ctx, payload.ID)
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

		var messageNotif []string
		newBoardData := new(model.BoardEntityModel)
		newBoardData.Context = ctx
		newBoardData.ID = payload.ID
		if payload.Name != nil {
			newBoardData.Name = *payload.Name
			if boardData.Name != newBoardData.Name {
				messageNotif = append(messageNotif, fmt.Sprintf("Board berganti nama menjadi %s", newBoardData.Name))
			}
		}
		if payload.WorkspaceId != nil {
			newWorkspaceData, err := s.WorkspaceRepository.FindById(ctx, *payload.WorkspaceId)
			if err != nil && err.Error() != "record not found" {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
			if newWorkspaceData == nil {
				return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "workspace not found")
			}

			dataBoardInWorkspace, err := s.BoardRepository.FindByWorkspaceId(ctx, newWorkspaceData.ID)
			if err != nil && err.Error() != "record not found" {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}

			sortNum := 1
			if dataBoardInWorkspace != nil {
				sortNum = dataBoardInWorkspace.SortNumber + 1
			}

			newBoardData.WorkspaceId = newWorkspaceData.ID
			newBoardData.SortNumber = sortNum
			if workspaceData.ID != newWorkspaceData.ID {
				messageNotif = append(messageNotif, fmt.Sprintf("Board dipindahkan ke %s", workspaceData.Name))
			}
		}
		if payload.SortNumber != nil {
			newBoardData.SortNumber = *payload.SortNumber
		}

		if err = s.BoardRepository.Update(ctx, newBoardData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		for _, v := range userAdmin {
			if v.ID != userLogin.ID {
				for _, d := range messageNotif {
					modelNotifikasi := new(model.NotifikasiEntityModel)
					modelNotifikasi.Context = ctx
					modelNotifikasi.Title = fmt.Sprintf("%s telah mengupdate board %s di %s", userLogin.Name, boardData.Name, workspaceData.Name)
					modelNotifikasi.Message = d
					modelNotifikasi.IsRead = false
					modelNotifikasi.UserId = v.ID
					modelNotifikasi.TaskId = constant.BLANK_TASK_ID
					if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
						return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
					}

					if err := ws.PublishNotificationWithoutTransaction(v.ID, s.DB, ctx); err != nil {
						return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
					}
				}
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

func (s *service) FindByWorkspaceId(ctx *abstraction.Context, payload *dto.BoardFindByWorkspaceIDRequest) (map[string]interface{}, error) {
	var res []map[string]interface{} = nil

	workspaceData, err := s.WorkspaceRepository.FindById(ctx, payload.WorkspaceID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	if workspaceData == nil {
		return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "workspace not found")
	}

	data, err := s.BoardRepository.FindByWorkspaceIdArr(ctx, payload.WorkspaceID, false)
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
			"created_at":   general.FormatWithZWithoutChangingTime(v.CreatedAt),
			"updated_at":   general.FormatWithZWithoutChangingTime(*v.UpdatedAt),
		})
	}

	return map[string]interface{}{
		"count": count,
		"data":  res,
	}, nil
}
