package main

import (
	"context"
	"net/http"
	"time"

	"github.com/mateconpizza/goairdrop/internal/application"
	"github.com/mateconpizza/goairdrop/internal/cli"
	"github.com/mateconpizza/goairdrop/internal/server"
	"github.com/mateconpizza/goairdrop/internal/server/cleanup"
	"github.com/mateconpizza/goairdrop/internal/server/middleware"
	"github.com/mateconpizza/goairdrop/internal/webui"
)

const (
	appName = "goaird"
	version = "0.1.1"
)

func main() {
	app := application.New(appName, version)
	if err := app.Init(); err != nil {
		cli.ErrAndExit(app.Name, err)
	}

	if err := run(app); err != nil {
		cli.ErrAndExit(app.Name, err)
	}
}

func run(app *application.App) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := http.NewServeMux()
	mux, err := app.Routes(mux)
	if err != nil {
		return err
	}

	ui, err := webui.New(app)
	if err != nil {
		return err
	}
	ui.Routes(mux)

	srv := server.New(
		server.WithMux(mux),
		server.WithAddr(app.Cfg.Server.Addr),
		server.WithLogger(app.Logger),
		server.WithMiddleware([]server.Middleware{
			middleware.Logging,
			middleware.PanicRecover,
		}...),
	)

	cleanup.Register(
		func() error {
			app.Logger.Info("shutting down HTTP server")
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer shutdownCancel()

			return srv.Shutdown(shutdownCtx)
		},
	)
	cleanup.Listen(ctx, cancel, app.Logger)

	return srv.Start()
}
