package main

import (
	"log"
	"log/slog"

	"github.com/mateconpizza/goairdrop/internal/application"
	"github.com/mateconpizza/goairdrop/internal/ui"
	"github.com/mateconpizza/goairdrop/internal/webhook"
)

const (
	Name    = "goairdrop"
	Version = "0.1.1"
)

func mainNew() {
	app := application.New(Name, Version)
	if err := app.Parse(); err != nil {
		log.Fatal(err)
	}

	if app.LogFile != nil {
		defer func() {
			if err := app.LogFile.Close(); err != nil {
				slog.Error("Failed closing log file", slog.String("error", err.Error()))
			}
		}()
	}

	if err := app.Run(); err != nil {
		app.Error(err)
		return
	}
}

func main() {
	app := application.New(Name, Version)
	if err := app.Parse(); err != nil {
		log.Fatal(err)
	}

	defer func() {
		if err := app.LogFile.Close(); err != nil {
			slog.Error("Failed closing log file", slog.String("error", err.Error()))
		}
	}()

	if err := app.LoadConfig(); err != nil {
		slog.Error("main", slog.String("error", err.Error()))
		return
	}

	mux, err := app.SetupRoutes()
	if err != nil {
		slog.Error("main", slog.String("error", err.Error()))
		return
	}

	uiHandler, err := ui.New(app)
	if err != nil {
		slog.Error("main", slog.String("error", err.Error()))
		return
	}
	uiHandler.SetupRoutes(mux)

	server := webhook.New(app.CmdArgs.Addr, mux)

	slog.Info("Server starting on " + app.CmdArgs.Addr)
	if err := server.Start(); err != nil {
		slog.Error("main", slog.String("error", err.Error()))
		return
	}
}
