// Package webhook...
package hook

import (
	"errors"
	"fmt"
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
	ErrHooksEmpty           = errors.New("no hooks registered")

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

func (h *Hook) String() string {
	if h == nil {
		return "<nil Hook>"
	}

	var sb strings.Builder
	typeColor := cli.Green
	if h.Type == TypeCommand {
		typeColor = cli.Red
	}
	htype := typeColor.Wrap("["+string(h.Type)+"]", cli.Bold)

	// Header: Hook[type] "Name" (METHOD /endpoint)
	fmt.Fprintf(&sb, "Hook%s %q (%s %s)", htype, h.Name, h.Method, h.Endpoint)

	// Status badges
	if h.Disabled {
		sb.WriteString(cli.Red.Sprint(" [disabled]"))
	}
	if h.Notify {
		sb.WriteString(cli.Blue.Sprint(" [notify]"))
	}

	// Type-specific details
	switch h.Type {
	case TypeUpload:
		if h.Destination != "" {
			fmt.Fprintf(&sb, "\n  destination: %s", h.Destination)
		}
		if h.MaxSizeMB > 0 {
			fmt.Fprintf(&sb, "\n  max_size:    %d MB", h.MaxSizeMB)
		}
		if h.FilenameStrategy != "" {
			fmt.Fprintf(&sb, "\n  filename:    %s", h.FilenameStrategy)
		}
		if len(h.AllowedMIMETypes) > 0 {
			fmt.Fprintf(&sb, "\n  mime_types:  %s", strings.Join(h.AllowedMIMETypes, ", "))
		}

	case TypeCommand:
		if h.CommandTemplate != nil {
			c := h.CommandTemplate
			fmt.Fprintf(&sb, "\n  command:     %s %s", c.Command, strings.Join(c.Args, " "))
			if c.TimeoutSeconds > 0 {
				fmt.Fprintf(&sb, "\n  timeout:     %ds", c.TimeoutSeconds)
			}
		}
		if len(h.AllowedActions) > 0 {
			fmt.Fprintf(&sb, "\n  actions:     %s", strings.Join(h.AllowedActions, ", "))
		}
	}

	if h.RateLimitPerMinute > 0 {
		fmt.Fprintf(&sb, "\n  rate_limit:  %d/min", h.RateLimitPerMinute)
	}

	return sb.String()
}

type Manager struct {
	appName string
	logger  *slog.Logger
	Hooks   []*Hook
}

func (m *Manager) Register(h *Hook) {
	m.Hooks = append(m.Hooks, h)
}

func (m *Manager) Find(name string) (*Hook, error) {
	if len(m.Hooks) == 0 {
		return nil, ErrHooksEmpty
	}

	for i := range m.Hooks {
		h := m.Hooks[i]
		if h.Name == name || h.Endpoint == name {
			return h, nil
		}
	}

	return nil, fmt.Errorf("%w: %q", ErrHookNotFound, name)
}

func NewManager(appName string, logger *slog.Logger) *Manager {
	return &Manager{appName: appName, logger: logger}
}

func (m *Manager) PrettifyHooks() string {
	slices.SortFunc(m.Hooks, func(a, b *Hook) int {
		return strings.Compare(string(a.Type), string(b.Type))
	})

	headers := []string{"Name", "Type", "Method", "Endpoint", "Dest", "Enabled"}
	rows := make([][]string, 0, len(m.Hooks))
	footer := []string{}

	for i := range m.Hooks {
		h := m.Hooks[i]
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
				c.Sprint(string(h.Type)),
				h.Method,
				h.Endpoint,
				dest,
				enabled.Sprint(strconv.FormatBool(!h.Disabled)),
			},
		)
	}

	return cli.Table(headers, rows, footer...)
}
