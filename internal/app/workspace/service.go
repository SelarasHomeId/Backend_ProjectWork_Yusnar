package workspace

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/dto"
	"selarashomeid/internal/factory"
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/gdrive"
	"selarashomeid/pkg/util/general"
	"selarashomeid/pkg/util/response"
	"strconv"

	"github.com/go-redis/redis/v8"
	"google.golang.org/api/drive/v3"
	"gorm.io/gorm"
)

type Service interface {
	Find(ctx *abstraction.Context) (map[string]interface{}, error)
	GetData(ctx *abstraction.Context, payload *dto.WorkspaceGetDataRequest) (map[string]interface{}, error)
}

type service struct {
	WorkspaceRepository     repository.Workspace
	BoardRepository         repository.Board
	TaskRepository          repository.Task
	TaskFileRepository      repository.TaskFile
	TaskCommentRepository   repository.TaskComment
	TaskLabelRepository     repository.TaskLabel
	ChecklistItemRepository repository.ChecklistItem
	UserRepository          repository.User
	ProjectRepository       repository.Project

	DB      *gorm.DB
	DbRedis redis.Client
	sDrive  *drive.Service
	fDrive  *drive.File
}

func NewService(f *factory.Factory) Service {
	return &service{
		WorkspaceRepository:     f.WorkspaceRepository,
		BoardRepository:         f.BoardRepository,
		TaskRepository:          f.TaskRepository,
		TaskFileRepository:      f.TaskFileRepository,
		TaskCommentRepository:   f.TaskCommentRepository,
		TaskLabelRepository:     f.TaskLabelRepository,
		ChecklistItemRepository: f.ChecklistItemRepository,
		UserRepository:          f.UserRepository,
		ProjectRepository:       f.ProjectRepository,

		DB:      f.Db,
		DbRedis: *f.DbRedis,
		sDrive:  f.GDrive.Service,
		fDrive:  f.GDrive.Folder,
	}
}

func (s *service) Find(ctx *abstraction.Context) (map[string]interface{}, error) {
	var (
		res   []map[string]interface{} = nil
		count *int
	)

	data, err := s.WorkspaceRepository.Find(ctx)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	count, err = s.WorkspaceRepository.Count(ctx)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	for _, v := range data {
		projectData, err := s.ProjectRepository.FindById(ctx, v.ProjectId)
		if err != nil && err.Error() != "record not found" {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		workspace := map[string]interface{}{
			"id":         v.ID,
			"project_id": v.ProjectId,
			"name":       v.Name,
			"cover":      nil,
			"is_delete":  v.IsDelete,
			"created_at": v.CreatedAt,
			"updated_at": v.UpdatedAt,
		}

		if projectData.Cover != nil {
			cover, err := gdrive.GetFile(s.sDrive, *projectData.Cover)
			if err != nil {
				return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "cover not found")
			}
			workspace["cover"] = map[string]interface{}{
				"view":    "https://lh3.googleusercontent.com/d/" + *projectData.Cover,
				"content": cover.WebContentLink,
				"name":    cover.Name,
				"id":      cover.Id,
			}
		}

		res = append(res, workspace)
	}

	return map[string]interface{}{
		"count": count,
		"data":  res,
	}, nil
}

func (s *service) GetData(ctx *abstraction.Context, payload *dto.WorkspaceGetDataRequest) (map[string]interface{}, error) {
	var resBoard []map[string]interface{} = nil

	workspaceData, err := s.WorkspaceRepository.FindById(ctx, payload.ID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	if workspaceData == nil {
		return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "workspace not found")
	}

	boardData, err := s.BoardRepository.FindByWorkspaceIdArrNoLimitOrder(ctx, payload.ID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	boardCount, err := s.BoardRepository.CountByWorkspaceIdArrNoLimitOrder(ctx, payload.ID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}

	for _, board := range boardData {
		var resTask []map[string]interface{} = nil

		taskData, err := s.TaskRepository.FindByBoardIdArrNoLimitOrder(ctx, board.ID)
		if err != nil && err.Error() != "record not found" {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		taskCount, err := s.TaskRepository.CountByBoardIdArrNoLimitOrder(ctx, board.ID)
		if err != nil && err.Error() != "record not found" {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		for _, v := range taskData {
			countFileData, err := s.TaskFileRepository.CountByTaskId(ctx, v.ID)
			if err != nil && err.Error() != "record not found" {
				return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}

			countCommentData, err := s.TaskCommentRepository.CountByTaskId(ctx, v.ID)
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
				"file":           countFileData,
				"comment":        countCommentData,
				"checklist":      nil,
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
				"watch":       isWatch,
				"sort_number": v.SortNumber,
			}

			if v.AssignToUser != nil {
				var assignToUser []map[string]interface{}
				assignToUserArr := general.StringToArrayInt(*v.AssignToUser)
				for _, v := range assignToUserArr {
					dataUser, err := s.UserRepository.FindById(ctx, v)
					if err != nil && err.Error() != "record not found" {
						return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
					}
					if dataUser != nil {
						assignToUser = append(assignToUser, map[string]interface{}{
							"id":    dataUser.ID,
							"name":  dataUser.Name,
							"email": dataUser.Email,
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
					"view":    "https://lh3.googleusercontent.com/d/" + *v.Cover,
					"content": cover.WebContentLink,
					"name":    cover.Name,
					"id":      cover.Id,
				}
			}

			if v.Label != nil {
				var label []map[string]interface{}
				labelArr := general.StringToArrayInt(*v.Label)
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

			resTask = append(resTask, task)
		}

		board := map[string]interface{}{
			"id":          board.ID,
			"name":        board.Name,
			"task_total":  board.TaskTotal,
			"sort_number": board.SortNumber,
			"task": map[string]interface{}{
				"data":  resTask,
				"count": taskCount,
			},
		}

		resBoard = append(resBoard, board)
	}

	resWorkspace := map[string]interface{}{
		"id":        workspaceData.ID,
		"workspace": workspaceData.Name,
		"board": map[string]interface{}{
			"data":  resBoard,
			"count": boardCount,
		},
	}

	return map[string]interface{}{
		"data": resWorkspace,
	}, nil
}
