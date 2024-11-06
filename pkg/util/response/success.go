package response

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func SuccessBuilder(code int, data interface{}) *MetaSuccess {
	return &MetaSuccess{
		Success: true,
		Data:    data,
		Code:    code,
	}
}

func SuccessResponse(data interface{}) *MetaSuccess {
	return SuccessBuilder(http.StatusOK, data)
}

func (m *MetaSuccess) SendSuccess(c echo.Context) error {
	return c.JSON(m.Code, m)
}
