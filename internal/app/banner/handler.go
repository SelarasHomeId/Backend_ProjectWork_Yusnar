package banner

import (
	"selarashomeid/internal/abstraction"
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

func (h handler) Find(c echo.Context) (err error) {
	data, err := h.service.Find(c.(*abstraction.Context))
	if err != nil {
		return response.ErrorResponse(err).SendError(c)
	}
	return response.SuccessResponse(data).SendSuccess(c)
}
