package http

import (
	"fmt"
	"net/http"

	_ "selarashomeid/docs"
	"selarashomeid/internal/app/auth"
	"selarashomeid/internal/app/banner"
	"selarashomeid/internal/app/board"
	"selarashomeid/internal/app/crm"
	"selarashomeid/internal/app/divisi"
	"selarashomeid/internal/app/notifikasi"
	"selarashomeid/internal/app/project"
	"selarashomeid/internal/app/role"
	"selarashomeid/internal/app/task"
	"selarashomeid/internal/app/test"
	user "selarashomeid/internal/app/user"
	"selarashomeid/internal/app/workspace"
	"selarashomeid/internal/config"
	"selarashomeid/internal/factory"
	"selarashomeid/pkg/constant"

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

	// static
	e.Static("/images", constant.PATH_ASSETS_IMAGES)
	e.Static("/share", constant.PATH_SHARE)
	e.Static("/file_saved", constant.PATH_FILE_SAVED)

	// routes
	test.NewHandler(f).Route(e.Group("/test"))
	banner.NewHandler(f).Route(e.Group("/banner"))
	auth.NewHandler(f).Route(e.Group("/auth"))
	user.NewHandler(f).Route(e.Group("/user"))
	role.NewHandler(f).Route(e.Group("/role"))
	divisi.NewHandler(f).Route(e.Group("/divisi"))
	notifikasi.NewHandler(f).Route(e.Group("/notifikasi"))
	workspace.NewHandler(f).Route(e.Group("/workspace"))
	board.NewHandler(f).Route(e.Group("/board"))
	task.NewHandler(f).Route(e.Group("/task"))
	crm.NewHandler(f).Route(e.Group("/crm"))
	project.NewHandler(f).Route(e.Group("/project"))
}
