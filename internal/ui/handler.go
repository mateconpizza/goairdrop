// Package ui...
package ui

import (
	"embed"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/mateconpizza/goairdrop/internal/application"
)

//go:embed templates/*.gohtml
var templateFS embed.FS

type Handler struct {
	tmpl *template.Template
	app  *application.App
}

func New(app *application.App) (*Handler, error) {
	entries, _ := templateFS.ReadDir("templates")
	for _, e := range entries {
		slog.Info("embedded file", slog.String("name", e.Name()))
	}

	tmpl, err := template.ParseFS(templateFS, "templates/*.gohtml")
	if err != nil {
		return nil, err
	}

	return &Handler{tmpl: tmpl, app: app}, nil
}

func (h *Handler) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /config", h.showConfig)
	mux.HandleFunc("POST /config", h.saveConfig)
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
