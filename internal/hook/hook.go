// Package webhook...
package hook

import (
	"errors"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"github.com/mateconpizza/goairdrop/internal/cli"
)

var (
	ErrHookEndpointRequired = errors.New("endpoint is required")
	ErrHookMethodRequired   = errors.New("method is required")
	ErrHookNameRequired     = errors.New("name is required")
	ErrHookNotFound         = errors.New("hook not found")
	ErrHookTypeRequired     = errors.New("hoot type is required (upload|command)")
	ErrHookUnknownType      = errors.New("hook unknown type")
	ErrHookEndpointInvalid  = errors.New("hook endpoint invalid")

	// Command.
	ErrHookCmdTemplateCmdRequired = errors.New("command_template.command required")
	ErrHookCmdTemplateRequired    = errors.New("command_template required for command")

	// Upload.
	ErrHookUploadDestRequired = errors.New("destination required")
)

type HookType string

const (
	TypeCommand HookType = "command"
	TypeUpload  HookType = "upload"
)

type Hook struct {
	Name               string   `json:"name"`
	Type               HookType `json:"type"`
	Endpoint           string   `json:"endpoint"`
	Method             string   `json:"method"`
	Destination        string   `json:"destination,omitempty"`
	MaxSizeMB          int      `json:"max_size_mb,omitempty"`
	RateLimitPerMinute int      `json:"rate_limit_per_minute,omitempty"`
	FilenameStrategy   string   `json:"filename_strategy,omitempty"`
	AllowedMIMETypes   []string `json:"allowed_mime_types,omitempty"`

	CommandTemplate *ExecConfig `json:"command_template,omitempty"`
	AllowedActions  []string    `json:"allowed_actions,omitempty"`

	Notify   bool `json:"notify"`
	Disabled bool `json:"disabled"`
}

type ExecConfig struct {
	Command        string   `json:"command"`
	Args           []string `json:"args"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}

type Manager struct {
	appName string
	logger  *slog.Logger
}

func NewManager(appName string, logger *slog.Logger) *Manager {
	return &Manager{appName: appName, logger: logger}
}

func (h *Hook) Validate() error {
	if h.Name == "" {
		return ErrHookNameRequired
	}

	if h.Disabled {
		return nil
	}

	if h.Endpoint == "" {
		return ErrHookEndpointRequired
	}

	if !strings.HasPrefix(h.Endpoint, "/") {
		return ErrHookEndpointInvalid
	}

	if h.Method == "" {
		return ErrHookMethodRequired
	}

	switch h.Type {
	case TypeUpload:
		if h.Destination == "" {
			return ErrHookUploadDestRequired
		}

	case TypeCommand:
		if h.CommandTemplate == nil {
			return ErrHookCmdTemplateRequired
		}

		if h.CommandTemplate.Command == "" {
			return ErrHookCmdTemplateCmdRequired
		}

	default:
		return ErrHookTypeRequired
	}

	return nil
}

func PrettyHooks(hooks []Hook) string {
	slices.SortFunc(hooks, func(a, b Hook) int {
		return strings.Compare(string(a.Type), string(b.Type))
	})

	headers := []string{"Name", "Type", "Method", "Endpoint", "Dest", "Enabled"}
	rows := make([][]string, 0, len(hooks))
	footer := []string{}

	for i := range hooks {
		h := hooks[i]
		dest := h.Destination

		c := cli.Green
		if h.Type == TypeCommand {
			c = cli.Red
			dest = "-"
		}

		enabled := cli.Green
		if h.Disabled {
			enabled = cli.Red
		}

		rows = append(
			rows,
			[]string{
				h.Name,
				c(string(h.Type)),
				h.Method,
				h.Endpoint,
				dest,
				enabled(strconv.FormatBool(!h.Disabled)),
			},
		)
	}

	return cli.Table(headers, rows, footer...)
}
