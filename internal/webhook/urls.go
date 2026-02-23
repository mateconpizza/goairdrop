package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/mateconpizza/goairdrop/internal/cli"
	"github.com/mateconpizza/goairdrop/internal/notify"
)

// URLWebhookPayload represents the structure.
type URLWebhookPayload struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	Action  string `json:"action"`
}

// openOperation opens the URL in the default browser.
func openOperation(ctx context.Context, msg URLWebhookPayload) Response {
	resp := Response{
		Success: true,
		Message: "Opened text: " + msg.Content,
	}

	err := openURL(ctx, msg.Content)
	if err != nil {
		resp.Success = false
		resp.Message = "Error opening text: " + msg.Content
	}

	return resp
}

// HandlerURL handles the incoming webhook requests.
func HandlerURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	clientIP := getClientIP(r)

	slog.Info("Received request",
		slog.String("ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
		slog.String("path", r.URL.Path),
	)

	var msg URLWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Error decoding JSON", http.StatusBadRequest)
		slog.Error("Error decoding JSON", slog.String("error", err.Error()))
		return
	}

	slog.Info("Received action", slog.String("action", msg.Action), slog.String("ip", clientIP))

	var resp Response
	switch msg.Action {
	case "open":
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(5*time.Second))
		defer cancel()
		resp = openOperation(ctx, msg)
	default:
		resp = Response{Success: false, Message: "Unknown action: %s" + msg.Action}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Error encoding response", slog.String("error", err.Error()))
	}

	slog.Info("Sent response",
		slog.Bool("success", resp.Success),
		slog.String("message", resp.Message),
	)
}

// openURL opens a URL in the default browser.
func openURL(ctx context.Context, s string) error {
	args := cli.OSArgs()
	if err := cli.ExecuteCmd(ctx, append(args, s)...); err != nil {
		return fmt.Errorf("%w: opening in browser", err)
	}

	t := notify.New(
		notify.WithContext(ctx),
		notify.WithBody("Opening URL: "+s),
		notify.WithAppName("goaird"),
		notify.WithIcon(notify.IconInfo),
		notify.WithUrgency(notify.UrgencyNormal),
		notify.WithID(999),
	)

	if _, err := t.Send(); err != nil {
		slog.Error("Failed to send notification", slog.String("error", err.Error()))
	}

	return nil
}
