package webhook

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

var ErrServerAlreadyRunning = errors.New("server is already running")

// Response represents the structure of the outgoing messages.
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Webhook represents the webhook server.
type Webhook struct {
	*http.Server
	isActive int32 // use atomic operations
}

// New creates a new Webhook instance with custom settings and handler.
func New(addr string, handler http.Handler) *Webhook {
	return &Webhook{
		Server: &http.Server{
			Addr:         addr,
			Handler:      handler,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

// Start starts the HTTP server.
func (w *Webhook) Start() error {
	if !atomic.CompareAndSwapInt32(&w.isActive, 0, 1) {
		return ErrServerAlreadyRunning
	}

	return w.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (w *Webhook) Shutdown(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&w.isActive, 0, 1) {
		slog.Debug("Server is not running")
		return nil
	}

	return w.Server.Shutdown(ctx)
}

// SetLogger sets the logger for the webhook.
// func SetLogger(l *slog.Logger) {
// 	logger = l
// }

// getClientIP returns the IP address of the client.
func getClientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.SplitN(forwarded, ",", 2)[0]
	}

	return r.RemoteAddr
}

// getDeviceName extracts the device name from request headers or form values.
func getDeviceName(r *http.Request) string {
	deviceName := r.Header.Get("X-Device-Name")
	if deviceName == "" {
		deviceName = r.FormValue("device_name")
	}
	if deviceName == "" {
		deviceName = "unknown"
	}

	return deviceName
}
