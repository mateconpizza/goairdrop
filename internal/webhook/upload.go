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

// HandleUploadHook handles an HTTP POST with one or more uploaded files.
func HandleUploadHook(h *Hook) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		files, err := parseFiles(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		deviceName := getDeviceName(r)
		uploadDir := cli.ExpandUser(h.Destination)

		for _, fh := range files {
			if err := saveUploadedFile(fh, uploadDir); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		if h.Notify {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			notification := notify.New(
				notify.WithContext(ctx),
				notify.WithBody(
					fmt.Sprintf(
						"Received %d file(s) from device: %s\n",
						len(files),
						deviceName+" ("+getClientIP(r)+")",
					),
				),
				notify.WithAppName("goaird"),
				notify.WithIcon(notify.IconInfo),
				notify.WithUrgency(notify.UrgencyNormal),
				notify.WithID(999),
			)

			if _, err := notification.Send(); err != nil {
				slog.Error("Failed to send notification", slog.String("error", err.Error()))
			}
		}

		resp := Response{
			Success: true,
			Message: fmt.Sprintf("Successfully uploaded %d file(s)", len(files)),
		}

		slog.Info(resp.Message)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			slog.Error("Error encoding response", slog.String("error", err.Error()))
		}
	}
}
