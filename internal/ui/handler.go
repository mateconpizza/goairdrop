// Package ui...
package ui

import (
	"embed"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/mateconpizza/goairdrop/internal/application"
)

//go:embed "templates" "static"
var files embed.FS

type Handler struct {
	tmpl *template.Template
	app  *application.App
}

func New(app *application.App) (*Handler, error) {
	entries, _ := files.ReadDir("templates")
	for _, e := range entries {
		slog.Info("embedded file", slog.String("name", e.Name()))
	}

	tmpl, err := template.ParseFS(files, "templates/*.gohtml")
	if err != nil {
		return nil, err
	}

	return &Handler{tmpl: tmpl, app: app}, nil
}

func (h *Handler) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /config", h.showConfig)
	mux.HandleFunc("POST /config", h.saveConfig)

	// static and cache files
	staticFS, err := fs.Sub(files, "static")
	if err != nil {
		panic(err)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
}

func (h *Handler) showConfig(w http.ResponseWriter, r *http.Request) {
	slog.Info("showConfig hit")
	if err := h.tmpl.ExecuteTemplate(w, "layout.gohtml", h.app.Cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) saveConfig(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// parse and update h.app.Cfg from r.Form
	// persist it
	http.Redirect(w, r, "/config", http.StatusSeeOther)
}
