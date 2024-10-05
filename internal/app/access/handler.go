package access

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/dto"
	"selarashomeid/internal/factory"
	"selarashomeid/pkg/util/response"

	"github.com/labstack/echo/v4"
)

type handler struct {
	service Service
}

func NewHandler(f *factory.Factory) *handler {
	return &handler{
		service: NewService(f),
	}
}

func (h *handler) Create(c echo.Context) (err error) {
	payload := new(dto.AccessCreateRequest)
	if err = c.Bind(payload); err != nil {
		return response.ErrorBuilder(&response.ErrorConstant.BadRequest, err).Send(c)
	}
	if err = c.Validate(payload); err != nil {
		return response.ErrorBuilder(&response.ErrorConstant.Validation, err).Send(c)
	}
	data, err := h.service.Create(c.(*abstraction.Context), payload)
	if err != nil {
		return response.ErrorResponse(err).Send(c)
	}
	return response.RedirectTo(c, *data.Module, data.Option)
}

func (h *handler) GetCount(c echo.Context) (err error) {
	data, err := h.service.GetCount(c.(*abstraction.Context))
	if err != nil {
		return response.ErrorResponse(err).Send(c)
	}
	return response.SuccessResponse(data).Send(c)
}
