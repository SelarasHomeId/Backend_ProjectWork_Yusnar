package contact

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/dto"
	"selarashomeid/internal/factory"
	"selarashomeid/internal/model"
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
	payload := new(dto.ContactCreateRequest)
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
	return response.SuccessResponse(data).Send(c)
}

func (h handler) Find(c echo.Context) (err error) {
	f := new(dto.ContactFilter)
	if err := c.Bind(f); err != nil {
		return response.ErrorBuilder(&response.ErrorConstant.BadRequest, err).Send(c)
	}
	p := new(abstraction.Pagination)
	if err := c.Bind(p); err != nil {
		return response.ErrorBuilder(&response.ErrorConstant.Validation, err).Send(c)
	}
	var (
		data []*model.ContactEntityModel
		info *abstraction.PaginationInfo
	)
	if data, info, err = h.service.Find(c.(*abstraction.Context), f, p); err != nil {
		return response.ErrorResponse(err).Send(c)
	}
	return response.SuccessResponse(data).WithPagination(info).Send(c)
}

func (h handler) FindByID(c echo.Context) (err error) {
	payload := new(dto.ContactFindByIDRequest)
	if err := c.Bind(payload); err != nil {
		return response.ErrorBuilder(&response.ErrorConstant.BadRequest, err).Send(c)
	}
	var data *model.ContactEntityModel
	if data, err = h.service.FindByID(c.(*abstraction.Context), payload); err != nil {
		return response.ErrorResponse(err).Send(c)
	}
	return response.SuccessResponse(data).Send(c)
}

func (h handler) DeleteByID(c echo.Context) (err error) {
	payload := new(dto.ContactDeleteByIDRequest)
	if err := c.Bind(payload); err != nil {
		return response.ErrorBuilder(&response.ErrorConstant.BadRequest, err).Send(c)
	}
	data, err := h.service.DeleteByID(c.(*abstraction.Context), payload)
	if err != nil {
		return response.ErrorResponse(err).Send(c)
	}
	return response.SuccessResponse(data).Send(c)
}

func (h handler) Export(c echo.Context) (err error) {
	filename, data, err := h.service.Export(c.(*abstraction.Context))
	if err != nil {
		return response.ErrorResponse(err).Send(c)
	}
	return response.SendExcelData(c, filename, *data)
}
