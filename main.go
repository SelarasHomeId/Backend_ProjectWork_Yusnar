package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"selarashomeid/internal/config"
	"selarashomeid/internal/factory"
	httpselarashomeid "selarashomeid/internal/http"
	middlewareEcho "selarashomeid/internal/middleware"
	db "selarashomeid/pkg/database"
	"selarashomeid/pkg/log"
	"selarashomeid/pkg/ngrok"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// @title selarashomeid
// @version 1.0.0
// @description This is a doc for selarashomeid.

func main() {
	config.Init()

	log.Init()

	db.Init()

	e := echo.New()

	f := factory.NewFactory()

	middlewareEcho.Init(e, f.DbRedis)

	httpselarashomeid.Init(e, f)

	ch := make(chan os.Signal, 1)

	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		runNgrok := false
		addr := ""
		if runNgrok {
			listener := ngrok.Run()
			e.Listener = listener
			addr = "/"
		} else {
			addr = ":" + config.Get().App.Port
		}
		err := e.Start(addr)
		if err != nil {
			if err != http.ErrServerClosed {
				logrus.Fatal(err)
			}
		}
	}()

	<-ch

	logrus.Println("Shutting down server...")
	cancel()

	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()
	e.Shutdown(ctx2)
	logrus.Println("Server gracefully stopped")
}
