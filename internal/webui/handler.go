// Package ui...
package webui

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/mateconpizza/goairdrop/internal/application"
	"github.com/mateconpizza/goairdrop/internal/server/cookie"
	"github.com/mateconpizza/goairdrop/internal/server/middleware"
)

//go:embed "templates" "static"
var files embed.FS

const idleTime = 20 * time.Minute // minutes

type Handler struct {
	tmpl     *template.Template
	app      *application.App
	data     *TemplateData
	cookies  *cookie.Jar
	sessions SessionStore
}

// TemplateData holds all data needed for template rendering.
type TemplateData struct {
	Cfg       *application.Config
	CSRFToken string
	Title     string
	ThemeMode string // light | dark
	AppName   string
	AppVer    string
	IsAuth    bool
}

func New(app *application.App) (*Handler, error) {
	entries, _ := files.ReadDir("templates")
	for _, e := range entries {
		app.Logger.Info("embedded file", "name", e.Name())
	}

	tmpl, err := template.ParseFS(files, "templates/*.gohtml")
	if err != nil {
		return nil, err
	}

	csrfToken, err := NewCSRFToken()
	if err != nil {
		return nil, err
	}

	f, _ := os.Open("/tmp/001/config.json")

	defer func() {
		if err := f.Close(); err != nil {
			app.Logger.Error("failed closing config file", "error", err)
		}
	}()

	var cfg application.Config
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	data := &TemplateData{
		Cfg:       &cfg,
		CSRFToken: csrfToken,
		AppName:   app.Name,
		AppVer:    app.Version,
		IsAuth:    false,
	}

	return &Handler{
		tmpl:     tmpl,
		app:      app,
		data:     data,
		cookies:  cookie.NewJar(app.Logger),
		sessions: NewMemoryStore(),
	}, nil
}

func (h *Handler) Routes(mux *http.ServeMux) {
	mux.Handle("/", middleware.Chain(http.HandlerFunc(h.index), localhostOnly))

	mux.Handle(
		"GET /config/",
		middleware.Chain(http.HandlerFunc(h.showConfig), localhostOnly),
	)

	mux.Handle(
		"POST /config/",
		middleware.Chain(http.HandlerFunc(h.saveConfig), localhostOnly, middleware.CSRFToken),
	)

	mux.Handle(
		"POST /theme/",
		middleware.Chain(http.HandlerFunc(h.toggleTheme), localhostOnly, middleware.CSRFToken),
	)

	mux.Handle("POST /auth", http.HandlerFunc(h.authPost))
	mux.Handle("POST /logout", http.HandlerFunc(h.logout))

	// static and cache files
	staticFS, err := fs.Sub(files, "static")
	if err != nil {
		panic(err)
	}

	mux.Handle(
		"GET /static/",
		http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))),
	)
}

func (h *Handler) index(w http.ResponseWriter, r *http.Request) {
	h.data.Title = "home"
	h.data.ThemeMode, _ = h.cookies.GetThemeMode(r)
	h.cookies.SetCSRFToken(w, h.data.CSRFToken)

	if err := h.tmpl.ExecuteTemplate(w, "index.gohtml", h.data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) showConfig(w http.ResponseWriter, r *http.Request) {
	h.data.Title = "configuration"
	h.data.ThemeMode, _ = h.cookies.GetThemeMode(r)
	h.cookies.SetCSRFToken(w, h.data.CSRFToken)

	if err := h.tmpl.ExecuteTemplate(w, "index.gohtml", h.data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) saveConfig(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("not implemented yet..."))
}

func (h *Handler) authPost(w http.ResponseWriter, r *http.Request) {
	const maxConfigSize = 64 << 10
	r.Body = http.MaxBytesReader(w, r.Body, maxConfigSize)

	h.data.Title = "auth"

	token := r.FormValue("token")

	if token == "" || token != h.app.Cfg.Server.Token {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if token == h.app.DefaultToken {
		h.app.Logger.Warn("request authenticated with default token",
			"method", r.Method,
			"path", r.URL.Path,
			"ip", r.RemoteAddr,
		)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	h.data.IsAuth = true
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) toggleTheme(w http.ResponseWriter, r *http.Request) {
	// read current theme from cookie
	current, _ := h.cookies.GetThemeMode(r)

	// toggle
	next := "dark"
	if current == "dark" {
		next = "light"
	}

	// set cookie
	h.cookies.SetThemeMode(w, next)

	// update data
	h.data.ThemeMode = next

	// redirect back
	http.Redirect(w, r, r.Referer(), http.StatusSeeOther)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	token, err := h.cookies.GetSession(r)
	if err == nil {
		_ = h.sessions.Delete(token)
	}

	h.cookies.ClearSession(w)
	h.cookies.ClearCSRF(w)
	h.data.IsAuth = false

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func newRandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

func NewSessionToken() (string, error) {
	return newRandomToken(32)
}

func NewCSRFToken() (string, error) {
	return newRandomToken(32)
}

func localhostOnly(next http.Handler) http.Handler {
	// NOTE: with reverse proxy (caddy)
	// - `RemoteAddr` will be always `127.0.0.1`
	// switch to:
	// - validating X-Forwarded-For
	// - or isolating /config on a separate port (maybe?)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// appendAt grows a slice to fit index j and sets the value.
func appendAt(s []string, j int, val string) []string {
	for len(s) <= j {
		s = append(s, "")
	}
	s[j] = val
	return s
}

type contextKey string

const userKey contextKey = "userID"

func (h *Handler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := h.cookies.GetSession(r)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		userID, err := h.sessions.Get(token)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) RotateSession(w http.ResponseWriter, r *http.Request) error {
	oldToken, err := h.cookies.GetSession(r)
	if err != nil {
		return err
	}

	user, err := h.sessions.Get(oldToken)
	if err != nil {
		return err
	}

	// borrar vieja
	_ = h.sessions.Delete(oldToken)

	// nueva
	newToken, _ := NewSessionToken()

	err = h.sessions.Create(user.UserID, newToken, time.Now().Add(cookie.Expiry24Hour))
	if err != nil {
		return err
	}

	h.cookies.SetSessionToken(w, newToken)

	return nil
}

func (h *Handler) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := h.cookies.GetSession(r)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		sess, err := h.sessions.Get(token)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// idle timeout
		if time.Since(sess.LastActive) > idleTime {
			_ = h.sessions.Delete(token)
			h.cookies.ClearSession(w)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// refresh actividad
		sess.LastActive = time.Now()
		_ = h.sessions.Update(token, sess)

		next.ServeHTTP(w, r)
	})
}
