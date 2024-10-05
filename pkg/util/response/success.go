package response

import (
	"bytes"
	"fmt"
	"net/http"
	"selarashomeid/internal/abstraction"
	"selarashomeid/pkg/constant"

	"github.com/labstack/echo/v4"
)

type successConstant struct {
	OK Success
}

var SuccessConstant successConstant = successConstant{
	OK: Success{
		Response: successResponse{
			Meta: Meta{
				Success: true,
				Message: "Request successfully proceed",
			},
			Data: nil,
		},
		Code: http.StatusOK,
	},
}

type successResponse struct {
	Meta Meta        `json:"meta"`
	Data interface{} `json:"data"`
}

type Success struct {
	Response successResponse `json:"response"`
	Code     int             `json:"code"`
}

func SuccessBuilder(res *Success, data interface{}) *Success {
	res.Response.Data = data
	return res
}

func CustomSuccessBuilder(code int, data interface{}, message string) *Success {
	return &Success{
		Response: successResponse{
			Meta: Meta{
				Success: true,
				Message: message,
			},
			Data: data,
		},
		Code: code,
	}
}

func SuccessResponse(data interface{}) *Success {
	return SuccessBuilder(&SuccessConstant.OK, data)
}

func (s *Success) Send(c echo.Context) error {
	return c.JSON(s.Code, s.Response)
}

func (s *Success) WithPagination(info *abstraction.PaginationInfo) *Success {
	return &Success{
		Response: successResponse{
			Meta: Meta{
				Success: s.Response.Meta.Success,
				Message: s.Response.Meta.Message,
				Info:    info,
			},
			Data: s.Response.Data,
		},
		Code: s.Code,
	}
}

func SendExcelData(c echo.Context, filename string, data bytes.Buffer) error {
	filename = filename + ".xlsx"
	c.Response().Header().Set(echo.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%s", filename))
	c.Response().Header().Set(echo.HeaderContentLength, fmt.Sprint(len(data.Bytes())))

	return c.Blob(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data.Bytes())
}

func RedirectTo(c echo.Context, module string, option *string) error {
	url := ""
	switch module {
	case "instagram":
		url = constant.LINK_INSTAGRAM
	case "tiktok":
		url = constant.LINK_TIKTOK
	case "facebook":
		url = constant.LINK_FACEBOOK
	case "whatsapp":
		if option != nil {
			url = constant.LINK_WHATSAPP + *option
		}
	}
	return c.Redirect(http.StatusFound, url)
}
