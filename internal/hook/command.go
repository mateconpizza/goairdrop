package hook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/mateconpizza/goairdrop/internal/cli"
	"github.com/mateconpizza/goairdrop/internal/notify"
)

func (m *Manager) NewCommand(h *Hook) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "bad JSON", http.StatusBadRequest)
			return
		}

		action, _ := payload["action"].(string)
		if !slices.Contains(h.AllowedActions, action) {
			m.logger.Error("commandHook", "action", action, "allowed", strings.Join(h.AllowedActions, ","))
			http.Error(w, "forbidden action", http.StatusForbidden)
			return
		}

		resolvedArgs := resolveTemplates(h.CommandTemplate.Args, payload)
		ctx, cancel := context.WithTimeout(
			context.Background(),
			time.Duration(h.CommandTemplate.TimeoutSeconds)*time.Second)
		defer cancel()

		command := cli.ExpandUser(h.CommandTemplate.Command)
		m.logger.Info("commandHook", "command", command)
		m.logger.Info("commandHook", "resolvedArgs", strings.Join(resolvedArgs, " "))
		cmd := exec.CommandContext(ctx, command, resolvedArgs...)
		err := cmd.Run()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		msg := fmt.Sprintf("%s %s", command, strings.Join(resolvedArgs, " "))
		if h.Notify {
			m.logger.Info("commandHook: notification to user", "message", msg)
			t := notify.New(
				notify.WithContext(ctx),
				notify.WithSummary(h.Name),
				notify.WithBody(msg),
				notify.WithAppName(m.appName),
				notify.WithIcon(notify.IconInfo),
				notify.WithUrgency(notify.UrgencyNormal),
				notify.WithID(999),
			)

			if _, err := t.Send(); err != nil {
				m.logger.Error("Failed to send notification", "error", err.Error())
			}
		}

		resp := Response{Success: true, Message: msg}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			m.logger.Error("Error encoding response", "error", err.Error())
		}
	}
}

func resolveTemplates(args []string, payload map[string]any) []string {
	resolved := make([]string, len(args))

	placeholderRE := regexp.MustCompile(`\{\{\s*payload\.([a-zA-Z0-9_]+)\s*\}\}`)

	for i, arg := range args {
		resolved[i] = placeholderRE.ReplaceAllStringFunc(arg, func(match string) string {
			submatches := placeholderRE.FindStringSubmatch(match)
			if len(submatches) != 2 {
				return ""
			}

			key := submatches[1]
			val, ok := payload[key]
			if !ok || val == nil {
				return ""
			}

			return stringify(val)
		})
	}

	return resolved
}

func stringify(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64: // JSON numbers decode as float64
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	default:
		return ""
	}
}
