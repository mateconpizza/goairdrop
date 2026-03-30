package cookie

import (
	"errors"
	"log/slog"
	"net/http"
	"time"
)

var ErrNotFound = errors.New("cookie not found")

const (
	Expiry24Hour = 24 * time.Hour
	Expiry1Hour  = 1 * time.Hour
)

const (
	session   = "session_token"
	csrf      = "csrf_token"
	ThemeMode = "theme_mode"
)

type Jar struct {
	logger *slog.Logger
}

func (j *Jar) SetSessionToken(w http.ResponseWriter, token string) {
	j.set(w, session, token, int(Expiry1Hour.Seconds()), true)
}

func (j *Jar) SetCSRFToken(w http.ResponseWriter, token string) {
	j.set(w, csrf, token, int(Expiry24Hour.Seconds()), false)
}

func (j *Jar) SetThemeMode(w http.ResponseWriter, next string) {
	j.set(w, ThemeMode, next, int(Expiry24Hour.Seconds()), false)
}

func (j *Jar) GetSession(r *http.Request) (string, error) {
	return j.get(r, session)
}

func (j *Jar) GetCSRF(r *http.Request) (string, error) {
	return j.get(r, csrf)
}

func (j *Jar) GetThemeMode(r *http.Request) (string, error) {
	return j.getWithDefault(r, ThemeMode, "light"), nil
}

func (j *Jar) ClearSession(w http.ResponseWriter) {
	j.logger.Debug("cookie: cleaning session token")
	j.clear(w, session)
}

func (j *Jar) ClearCSRF(w http.ResponseWriter) {
	// TODO: keep consistency in HttpOnly for CSRF???
	j.logger.Debug("cookie: cleaning CSRF token")
	j.clear(w, csrf)
}

func (j *Jar) set(w http.ResponseWriter, name, value string, maxAge int, httpOnly bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: httpOnly, // (<- false) Allow JavaScript access
		Secure:   true,     // Set to true HTTPS
		SameSite: http.SameSiteLaxMode,
	})
}

func (j *Jar) get(r *http.Request, name string) (string, error) {
	ck, err := r.Cookie(name)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return "", ErrNotFound
		}
		return "", err
	}
	return ck.Value, nil
}

func (j *Jar) clear(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (j *Jar) getWithDefault(r *http.Request, key, def string) string {
	ck, err := r.Cookie(key)
	if err != nil {
		j.logger.Debug("cookies: missing cookie, using default", "key", key, "default", def)
		return def
	}
	return ck.Value
}

func NewJar(logger *slog.Logger) *Jar {
	return &Jar{
		logger: logger,
	}
}
