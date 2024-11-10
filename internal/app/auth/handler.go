package auth

import (
	"net/http"
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
		return response.ErrorBuilder(http.StatusBadRequest, err, "error bind payload").SendError(c)
	}
	if err := c.Validate(payload); err != nil {
		return response.ErrorBuilder(http.StatusBadRequest, err, "error validate payload").SendError(c)
	}
	data, err := h.service.Login(c.(*abstraction.Context), payload)
	if err != nil {
		return response.ErrorResponse(err).SendError(c)
	}
	return response.SuccessResponse(data).SendSuccess(c)
}

func (h *handler) Logout(c echo.Context) error {
	data, err := h.service.Logout(c.(*abstraction.Context))
	if err != nil {
		return response.ErrorResponse(err).SendError(c)
	}
	return response.SuccessResponse(data).SendSuccess(c)
}

func (h *handler) RefreshToken(c echo.Context) error {
	data, err := h.service.RefreshToken(c.(*abstraction.Context))
	if err != nil {
		return response.ErrorResponse(err).SendError(c)
	}
	return response.SuccessResponse(data).SendSuccess(c)
}

func (h *handler) SendEmailResetPassword(c echo.Context) error {
	payload := new(dto.AuthSendEmailResetPasswordRequest)
	if err := c.Bind(payload); err != nil {
		return response.ErrorBuilder(http.StatusBadRequest, err, "error bind payload").SendError(c)
	}
	if err := c.Validate(payload); err != nil {
		return response.ErrorBuilder(http.StatusBadRequest, err, "error validate payload").SendError(c)
	}
	data, err := h.service.SendEmailResetPassword(c.(*abstraction.Context), payload)
	if err != nil {
		return response.ErrorResponse(err).SendError(c)
	}
	return response.SuccessResponse(data).SendSuccess(c)
}
