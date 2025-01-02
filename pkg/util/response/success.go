package response

import (
	"net/http"
	"selarashomeid/internal/dto"
	"selarashomeid/pkg/constant"

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

func RedirectTo(c echo.Context, data *dto.AccessCreateResponse) error {
	url := ""
	switch *data.Module {
	case "instagram":
		url = constant.LINK_INSTAGRAM
	case "tiktok":
		url = constant.LINK_TIKTOK
	case "facebook":
		url = constant.LINK_FACEBOOK
	case "whatsapp":
		url = constant.LINK_WHATSAPP
		if data.Phone != nil {
			url += *data.Phone
			if data.Message != nil {
				url += "?text=" + *data.Message
			}
		}
	}
	return c.Redirect(http.StatusFound, url)
}
