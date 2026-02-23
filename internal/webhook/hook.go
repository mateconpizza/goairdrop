// Package webhook...
package webhook

import (
	"errors"
	"fmt"
)

var (
	ErrHookNameRequired           = errors.New("name is required")
	ErrHookEndpointRequired       = errors.New("endpoint is required")
	ErrHookMethodRequired         = errors.New("method is required")
	ErrHookDestRequired           = errors.New("destination required for upload")
	ErrHookCmdTemplateRequired    = errors.New("command_template required for command")
	ErrHookCmdTemplateCmdRequired = errors.New("command_template.command required")
	ErrHookUnknownType            = errors.New("hook unknown type")
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

	Notify bool `json:"notify"`
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

	if h.Endpoint == "" {
		return fmt.Errorf("%w: %q", ErrHookEndpointRequired, h.Name)
	}

	if h.Method == "" {
		return fmt.Errorf("%w: %q", ErrHookMethodRequired, h.Name)
	}

	switch h.Type {
	case TypeUpload:
		if h.Destination == "" {
			return fmt.Errorf("%w: %q", ErrHookDestRequired, h.Name)
		}

	case TypeCommand:
		if h.CommandTemplate == nil {
			return fmt.Errorf("%w: %q", ErrHookCmdTemplateRequired, h.Name)
		}

		if h.CommandTemplate.Command == "" {
			return fmt.Errorf("%w: %q", ErrHookCmdTemplateCmdRequired, h.Name)
		}

	default:
		return fmt.Errorf("%w: %q name=%q", ErrHookUnknownType, h.Type, h.Name)
	}

	return nil
}
