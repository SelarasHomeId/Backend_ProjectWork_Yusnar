package http

import (
	"fmt"
	"net/http"

	_ "selarashomeid/docs"
	"selarashomeid/internal/app/auth"
	"selarashomeid/internal/app/divisi"
	"selarashomeid/internal/app/notifikasi"
	"selarashomeid/internal/app/role"
	"selarashomeid/internal/app/test"
	user "selarashomeid/internal/app/user"
	"selarashomeid/internal/factory"
	"selarashomeid/pkg/constant"

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

func Init(e *echo.Echo, f *factory.Factory) {
	var (
		APP     = constant.APP
		VERSION = constant.VERSION
	)

	// index
	e.GET("/api", func(c echo.Context) error {
		message := fmt.Sprintf("Hello there, welcome to api %s version %s.", APP, VERSION)
		return c.String(http.StatusOK, message)
	})

	// docs
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// routes
	api := e.Group("/api")
	test.NewHandler(f).Route(api.Group("/test"))
	auth.NewHandler(f).Route(api.Group("/auth"))
	user.NewHandler(f).Route(api.Group("/user"))
	role.NewHandler(f).Route(api.Group("/role"))
	divisi.NewHandler(f).Route(api.Group("/divisi"))
	notifikasi.NewHandler(f).Route(api.Group("/notifikasi"))
}
