package task

import (
	"net/http"
	"selarashomeid/internal/abstraction"
	taskchecklist "selarashomeid/internal/app/task/checklist"
	taskcomment "selarashomeid/internal/app/task/comment"
	taskfile "selarashomeid/internal/app/task/file"
	tasklabel "selarashomeid/internal/app/task/label"
	"selarashomeid/internal/dto"
	"selarashomeid/internal/factory"
	"selarashomeid/pkg/util/response"
	"strings"

	"github.com/labstack/echo/v4"
)

type handler struct {
	service Service

	TaskCommentHandler   taskcomment.Handler
	TaskFileHandler      taskfile.Handler
	TaskLabelHandler     tasklabel.Handler
	TaskChecklistHandler taskchecklist.Handler
}

func NewHandler(f *factory.Factory) *handler {
	return &handler{
		service: NewService(f),

		TaskCommentHandler:   *taskcomment.NewHandler(f),
		TaskFileHandler:      *taskfile.NewHandler(f),
		TaskLabelHandler:     *tasklabel.NewHandler(f),
		TaskChecklistHandler: *taskchecklist.NewHandler(f),
	}
}

func (h *handler) Create(c echo.Context) (err error) {
	payload := new(dto.TaskCreateRequest)
	if err = c.Bind(payload); err != nil {
		return response.ErrorBuilder(http.StatusBadRequest, err, "error bind payload").SendError(c)
	}
	if err = c.Validate(payload); err != nil {
		return response.ErrorBuilder(http.StatusBadRequest, err, "error validate payload").SendError(c)
	}
	data, err := h.service.Create(c.(*abstraction.Context), payload)
	if err != nil {
		return response.ErrorResponse(err).SendError(c)
	}
	return response.SuccessResponse(data).SendSuccess(c)
}

func (h handler) Delete(c echo.Context) (err error) {
	payload := new(dto.TaskDeleteByIDRequest)
	if err := c.Bind(payload); err != nil {
		return response.ErrorBuilder(http.StatusBadRequest, err, "error bind payload").SendError(c)
	}
	if err = c.Validate(payload); err != nil {
		return response.ErrorBuilder(http.StatusBadRequest, err, "error validate payload").SendError(c)
	}
	data, err := h.service.Delete(c.(*abstraction.Context), payload)
	if err != nil {
		return response.ErrorResponse(err).SendError(c)
	}
	return response.SuccessResponse(data).SendSuccess(c)
}

func (h handler) FindByBoardId(c echo.Context) (err error) {
	payload := new(dto.TaskFindByBoardIDRequest)
	if err := c.Bind(payload); err != nil {
		return response.ErrorBuilder(http.StatusBadRequest, err, "error bind payload").SendError(c)
	}
	if err = c.Validate(payload); err != nil {
		return response.ErrorBuilder(http.StatusBadRequest, err, "error validate payload").SendError(c)
	}
	data, err := h.service.FindByBoardId(c.(*abstraction.Context), payload)
	if err != nil {
		return response.ErrorResponse(err).SendError(c)
	}
	return response.SuccessResponse(data).SendSuccess(c)
}

func (h handler) Update(c echo.Context) (err error) {
	payload := new(dto.TaskUpdateRequest)

	if err = c.Bind(payload); err != nil {
		return response.ErrorBuilder(http.StatusBadRequest, err, "error bind payload").SendError(c)
	}
	if err = c.Validate(payload); err != nil {
		return response.ErrorBuilder(http.StatusBadRequest, err, "error validate payload").SendError(c)
	}

	contentType := c.Request().Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := c.Request().ParseMultipartForm(64 << 20); err != nil {
			return response.ErrorBuilder(http.StatusBadRequest, err, "error bind multipart/form-data").SendError(c)
		}
		payload.Cover = c.Request().MultipartForm.File["cover"]
	}

	data, err := h.service.Update(c.(*abstraction.Context), payload)
	if err != nil {
		return response.ErrorResponse(err).SendError(c)
	}
	return response.SuccessResponse(data).SendSuccess(c)
}

func (h handler) FindById(c echo.Context) (err error) {
	payload := new(dto.TaskFindByIDRequest)
	if err := c.Bind(payload); err != nil {
		return response.ErrorBuilder(http.StatusBadRequest, err, "error bind payload").SendError(c)
	}
	if err = c.Validate(payload); err != nil {
		return response.ErrorBuilder(http.StatusBadRequest, err, "error validate payload").SendError(c)
	}
	data, err := h.service.FindById(c.(*abstraction.Context), payload)
	if err != nil {
		return response.ErrorResponse(err).SendError(c)
	}
	return response.SuccessResponse(data).SendSuccess(c)
}

func (h handler) Find(c echo.Context) (err error) {
	data, err := h.service.Find(c.(*abstraction.Context))
	if err != nil {
		return response.ErrorResponse(err).SendError(c)
	}
	return response.SuccessResponse(data).SendSuccess(c)
}
