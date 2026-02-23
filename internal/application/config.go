package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/mateconpizza/goairdrop/internal/webhook"
)

const configFileName = "config.json"

var ErrServerAddrRequired = errors.New("server.address is required")

type Config struct {
	Server ServerConfig   `json:"server"`
	Hooks  []webhook.Hook `json:"hooks"`
}

type ServerConfig struct {
	Address             string `json:"address"`
	ReadTimeoutSeconds  int    `json:"read_timeout_seconds"`
	WriteTimeoutSeconds int    `json:"write_timeout_seconds"`
}

type HookOld struct {
	Endpoint         string   `json:"endpoint"`
	Method           string   `json:"method"`
	Destination      string   `json:"destination"`
	MaxSizeMB        int      `json:"max_size_mb"`
	AllowedMIMETypes []string `json:"allowed_mime_types"`
	RequireAuth      bool     `json:"require_auth"`
	Overwrite        bool     `json:"overwrite"`
	LogRequests      bool     `json:"log_requests"`
}

func (c *Config) Validate() error {
	if c.Server.Address == "" {
		return ErrServerAddrRequired
	}

	return nil
}

func parse(path string) (*Config, error) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		slog.Warn("config not found, using defaults", slog.String("path", path))
		return defaultConfig(), nil
	}

	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}

	defer func() {
		if err := f.Close(); err != nil {
			slog.Error("failed closing config file", slog.String("error", err.Error()))
		}
	}()

	var cfg Config
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	return &cfg, nil
}

func LoadOld(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}

	defer func() {
		if err := f.Close(); err != nil {
			slog.Error("Failed closing log file", slog.String("error", err.Error()))
		}
	}()

	var cfg Config

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	return &cfg, nil
}

func configPath(appName string) (string, error) {
	dirConfig, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dirConfig, appName, configFileName), nil
}

func defaultConfig() *Config {
	return &Config{
		Hooks: []webhook.Hook{
			{
				Name:     "default-url",
				Type:     webhook.TypeCommand,
				Endpoint: "/url",
				Method:   "POST",
				CommandTemplate: &webhook.ExecConfig{
					Command:        "xdg-open",
					Args:           []string{"{{payload.url}}"},
					TimeoutSeconds: 10,
				},
				AllowedActions: []string{"open"},
				Notify:         true,
			},
			{
				Name:               "default-files",
				Type:               webhook.TypeUpload,
				Endpoint:           "/files",
				Method:             "POST",
				RateLimitPerMinute: 30,
				MaxSizeMB:          50,
				Destination:        "~/dls/goairdrop/upload",
				FilenameStrategy:   "original",
				Notify:             true,
			},
		},
	}
}
