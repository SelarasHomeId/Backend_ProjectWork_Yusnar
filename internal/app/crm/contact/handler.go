package contact

import (
	"net/http"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/dto"
	"selarashomeid/internal/factory"
	"selarashomeid/pkg/util/response"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	service Service
}

func NewHandler(f *factory.Factory) *Handler {
	return &Handler{
		service: NewService(f),
	}
}

func (h *Handler) Create(c echo.Context) (err error) {
	payload := new(dto.ContactCreateRequest)
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

func (h Handler) Find(c echo.Context) (err error) {
	data, err := h.service.Find(c.(*abstraction.Context))
	if err != nil {
		return response.ErrorResponse(err).SendError(c)
	}
	return response.SuccessResponse(data).SendSuccess(c)
}

func (h Handler) Export(c echo.Context) (err error) {
	filename, data, err := h.service.Export(c.(*abstraction.Context))
	if err != nil {
		return response.ErrorResponse(err).SendError(c)
	}
	return response.SendExcelData(c, filename, *data)
}
