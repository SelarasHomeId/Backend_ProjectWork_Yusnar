package auth

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

func (h *handler) Login(c echo.Context) error {
	payload := new(dto.AuthLoginRequest)
	if err := c.Bind(payload); err != nil {
		return response.ErrorBuilder(&response.ErrorConstant.BadRequest, err).Send(c)
	}
	if err := c.Validate(payload); err != nil {
		return response.ErrorBuilder(&response.ErrorConstant.BadRequest, err).Send(c)
	}
	data, err := h.service.Login(c.(*abstraction.Context), payload)
	if err != nil {
		return response.ErrorResponse(err).Send(c)
	}
	return response.SuccessResponse(data).Send(c)
}

func (h *handler) Logout(c echo.Context) error {
	data, err := h.service.Logout(c.(*abstraction.Context))
	if err != nil {
		return response.ErrorResponse(err).Send(c)
	}
	return response.SuccessResponse(data).Send(c)
}

func (h *handler) ChangePassword(c echo.Context) error {
	cc := c.(*abstraction.Context)
	payload := new(dto.ChangePasswordRequest)
	if err := c.Bind(payload); err != nil {
		return response.ErrorBuilder(&response.ErrorConstant.BadRequest, err).Send(c)
	}
	if err := c.Validate(payload); err != nil {
		return response.ErrorBuilder(&response.ErrorConstant.Validation, err).Send(c)
	}
	data, err := h.service.ChangePassword(cc, payload)
	if err != nil {
		return response.ErrorResponse(err).Send(c)
	}
	return response.SuccessResponse(data).Send(c)
}

func (h *handler) ResetPassword(c echo.Context) error {
	cc := c.(*abstraction.Context)
	payload := new(dto.ResetPasswordRequest)
	if err := c.Bind(payload); err != nil {
		return response.ErrorBuilder(&response.ErrorConstant.BadRequest, err).Send(c)
	}
	if err := c.Validate(payload); err != nil {
		return response.ErrorBuilder(&response.ErrorConstant.Validation, err).Send(c)
	}
	data, err := h.service.ResetPassword(cc, payload)
	if err != nil {
		return response.ErrorResponse(err).Send(c)
	}
	return response.SuccessResponse(data).Send(c)
}
