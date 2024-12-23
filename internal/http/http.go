package http

import (
	"fmt"
	"net/http"

	_ "selarashomeid/docs"
	"selarashomeid/internal/app/auth"
	"selarashomeid/internal/app/banner"
	"selarashomeid/internal/app/divisi"
	"selarashomeid/internal/app/notifikasi"
	"selarashomeid/internal/app/role"
	"selarashomeid/internal/app/test"
	user "selarashomeid/internal/app/user"
	"selarashomeid/internal/config"
	"selarashomeid/internal/factory"

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

func Init(e *echo.Echo, f *factory.Factory) {
	// index
	e.GET("/", func(c echo.Context) error {
		message := fmt.Sprintf("Hello there, welcome to app %s version %s.", config.Get().App.App, config.Get().App.Version)
		return c.String(http.StatusOK, message)
	})

	// docs
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// routes
	test.NewHandler(f).Route(e.Group("/test"))
	auth.NewHandler(f).Route(e.Group("/auth"))
	user.NewHandler(f).Route(e.Group("/user"))
	role.NewHandler(f).Route(e.Group("/role"))
	divisi.NewHandler(f).Route(e.Group("/divisi"))
	notifikasi.NewHandler(f).Route(e.Group("/notifikasi"))
	banner.NewHandler(f).Route(e.Group("/banner"))
}
