package notification

import (
	"selarashomeid/internal/abstraction"
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

func (h handler) Find(c echo.Context) (err error) {
	var data []*model.NotificationEntityModel
	if data, err = h.service.Find(c.(*abstraction.Context)); err != nil {
		return response.ErrorResponse(err).Send(c)
	}
	return response.SuccessResponse(data).Send(c)
}

func (h handler) CountUnread(c echo.Context) (err error) {
	data, err := h.service.CountUnread(c.(*abstraction.Context))
	if err != nil {
		return response.ErrorResponse(err).Send(c)
	}
	return response.SuccessResponse(data).Send(c)
}

func (h handler) SetRead(c echo.Context) (err error) {
	payload := new(model.SetNotificationRead)
	if err = c.Bind(payload); err != nil {
		return response.ErrorBuilder(&response.ErrorConstant.BadRequest, err).Send(c)
	}
	if err = c.Validate(payload); err != nil {
		return response.ErrorBuilder(&response.ErrorConstant.Validation, err).Send(c)
	}
	data, err := h.service.SetRead(c.(*abstraction.Context), payload)
	if err != nil {
		return response.ErrorResponse(err).Send(c)
	}
	return response.SuccessResponse(data).Send(c)
}

func (h handler) FindByID(c echo.Context) (err error) {
	payload := new(model.NotificationFindByIDRequest)
	if err := c.Bind(payload); err != nil {
		return response.ErrorBuilder(&response.ErrorConstant.BadRequest, err).Send(c)
	}
	var data *model.NotificationEntityModel
	if data, err = h.service.FindByID(c.(*abstraction.Context), payload); err != nil {
		return response.ErrorResponse(err).Send(c)
	}
	return response.SuccessResponse(data).Send(c)
}
