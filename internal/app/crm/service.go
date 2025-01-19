package crm

import (
	"net/http"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/factory"
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/util/general"
	"selarashomeid/pkg/util/response"

	"gorm.io/gorm"
)

type Service interface {
	CalculateTask(ctx *abstraction.Context) (map[string]interface{}, error)
}

type service struct {
	TaskRepository      repository.Task
	BoardRepository     repository.Board
	WorkspaceRepository repository.Workspace

	DB *gorm.DB
}

func NewService(f *factory.Factory) Service {
	return &service{
		TaskRepository:      f.TaskRepository,
		BoardRepository:     f.BoardRepository,
		WorkspaceRepository: f.WorkspaceRepository,

		DB: f.Db,
	}
}

func (s *service) CalculateTask(ctx *abstraction.Context) (map[string]interface{}, error) {
	var res []map[string]interface{} = nil

	dataWorkspace, err := s.WorkspaceRepository.Find(ctx)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}

	for _, v := range dataWorkspace {
		dataBoard, err := s.BoardRepository.FindByWorkspaceIdArr(ctx, v.ID)
		if err != nil && err.Error() != "record not found" {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		var resBoard []map[string]interface{} = nil
		for _, b := range dataBoard {
			dataTask, err := s.TaskRepository.FindByBoardIdArr(ctx, b.ID)
			if err != nil && err.Error() != "record not found" {
				return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
			has_new := false
			for _, t := range dataTask {
				has_new = general.IsToday(t.CreatedAt)
			}
			resBoard = append(resBoard, map[string]interface{}{
				"name":       b.Name,
				"count_task": len(dataTask),
				"has_new":    has_new,
			})
		}

		res = append(res, map[string]interface{}{
			"workspace": v.Name,
			"board":     resBoard,
		})
	}

	return map[string]interface{}{
		"data": res,
	}, nil
}
