// Package middleware contains middleware functions for the HTTP server.
package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

const HeaderToken = "X-Goaird-Token" // #nosec G101

type Middleware func(http.Handler) http.Handler

type wrappedWriter struct {
	http.ResponseWriter
	statusCode int
}

func Chain(h http.Handler, m ...Middleware) http.Handler {
	for i := len(m) - 1; i >= 0; i-- {
		h = m[i](h)
	}
	return h
}

// Auth is a middleware that checks for a valid X-Goaird-Token.
func Auth(expectedToken, defaultToken string, next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get(HeaderToken)

		if token != expectedToken {
			logger.Warn("unauthorized request",
				"method", r.Method,
				"path", r.URL.Path,
				"ip", r.RemoteAddr,
			)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if token == defaultToken {
			logger.Warn("request authenticated with default token",
				"method", r.Method,
				"path", r.URL.Path,
				"ip", r.RemoteAddr,
			)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func PanicRecover(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered", "rec", rec, "stack", debug.Stack())
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func CSRFToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie("csrf_token")
		if err != nil {
			http.Error(w, "missing csrf cookie", http.StatusForbidden)
			return
		}

		const maxConfigSize = 64 << 10 // 64 KB
		r.Body = http.MaxBytesReader(w, r.Body, maxConfigSize)
		token := r.FormValue("csrf_token") // or header

		if token == "" || token != cookie.Value {
			http.Error(w, "invalid csrf token", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func Logging(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &wrappedWriter{
			ResponseWriter: w,
			statusCode:     0,
		}

		next.ServeHTTP(wrapped, r)

		if strings.HasPrefix(r.URL.Path, "/static/") || strings.HasPrefix(r.URL.Path, "/cache/favicon") {
			return
		}

		logger.Info(
			"request",
			"status", wrapped.statusCode,
			"method", r.Method,
			"path", strconv.Quote(r.URL.Path),
			"duration", time.Since(start),
		)
	})
}
