package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/mateconpizza/goairdrop/internal/cli"
	"github.com/mateconpizza/goairdrop/internal/hook"
)

const (
	configFileName = "config.json"

	defaultToken = "change-default-token-1234"
)

var (
	ErrServerAddrRequired  = errors.New("server.address is required")
	ErrServerTokenRequired = errors.New("server.token is required")
)

type Config struct {
	Server ServerConfig `json:"server"`
	Hooks  []hook.Hook  `json:"hooks"`
}

type ServerConfig struct {
	Addr  string `json:"address"`
	Token string `json:"token"`

	// FIX: Finish implementation
	ReadTimeoutSeconds  int `json:"read_timeout_seconds"`
	WriteTimeoutSeconds int `json:"write_timeout_seconds"`
}

func (c *Config) Validate() error {
	if c.Server.Addr == "" {
		return ErrServerAddrRequired
	}

	if c.Server.Token == "" {
		return ErrServerTokenRequired
	}

	return nil
}

func (c *Config) write(s string) error {
	return writeJSON(s, c)
}

func parse(app *App, path string) (*Config, error) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		app.Logger.Warn("config not found, using defaults", slog.String("path", path))
		return defaultConfig(app.Name), nil
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

func loadConfig(a *App) error {
	path, err := configPath(a.Name)
	if err != nil {
		return err
	}

	cfg, err := parse(a, path)
	if err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	a.Cfg = cfg
	a.CfgFile = path

	for i := range a.Cfg.Hooks {
		a.mgr.Register(&a.Cfg.Hooks[i])
	}

	return nil
}

func configPath(appName string) (string, error) {
	dirConfig, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dirConfig, appName, configFileName), nil
}

func defaultConfig(appName string) *Config {
	uploadDest := cli.XDGDataHome(appName)

	return &Config{
		Server: ServerConfig{
			Addr:  ":8080",
			Token: defaultToken,
		},
		Hooks: []hook.Hook{
			{
				Name:     "default-url",
				Type:     hook.TypeCommand,
				Endpoint: "/url",
				Method:   "POST",
				CommandTemplate: &hook.ExecConfig{
					Command:        "xdg-open",
					Args:           []string{"{{payload.url}}"},
					TimeoutSeconds: 10,
				},
				AllowedActions: []string{"open"},
				Notify:         true,
			},
			{
				Name:               "default-files",
				Type:               hook.TypeUpload,
				Endpoint:           "/files",
				Method:             "POST",
				RateLimitPerMinute: 30,
				MaxSizeMB:          50,
				Destination:        uploadDest,
				FilenameStrategy:   "original",
				Notify:             true,
			},
		},
	}
}

// func loadOrCreateToken(path string) (string, error) {
// 	if b, err := os.ReadFile(path); err == nil {
// 		return strings.TrimSpace(string(b)), nil
// 	}
//
// 	token := generateToken()
//
// 	err := os.WriteFile(path, []byte(token), FilePerm)
// 	if err != nil {
// 		return "", err
// 	}
//
// 	return token, nil
// }

// generateToken generates a 32-char string.
// func generateToken() string {
// 	b := make([]byte, 32) // 16 bytes = 128 bits of entropy
// 	if _, err := rand.Read(b); err != nil {
// 		return "fallback-please-change-me-12345"
// 	}
//
// 	return hex.EncodeToString(b)
// }
