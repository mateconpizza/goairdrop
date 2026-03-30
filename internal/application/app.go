// Package application...
package application

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/mateconpizza/goairdrop/internal/cli"
	"github.com/mateconpizza/goairdrop/internal/hook"
	"github.com/mateconpizza/goairdrop/internal/server/middleware"
)

type App struct {
	Name    string        // application name
	Version string        // application version
	cmd     string        // application subcommand
	Flag    *arguments    // parsed command-line arguments
	FlagSet *flag.FlagSet // command-line flag parser

	Cfg     *Config // application config
	CfgFile string  // config filepath

	Stdin  io.Reader // standard input
	Stdout io.Writer // standard output
	Stderr io.Writer // standard error

	Logger *slog.Logger // application logger

	DefaultToken string
}

type arguments struct {
	args    []string
	verbose bool
	list    bool
}

func (a *App) parse(c *commander) error {
	args := os.Args[1:]

	// no args → just flags
	if len(args) == 0 {
		return a.parseFlags(c, args)
	}

	// detect subcommand
	switch args[0] {
	case cmdGen:
		a.cmd = cmdGen
		a.Flag.args = args[1:]
		return nil
	case cmdVersion:
		a.cmd = cmdVersion
		return nil
	}

	// otherwise treat as flags
	return a.parseFlags(c, args)
}

func (a *App) parseFlags(c *commander, args []string) error {
	a.FlagSet.BoolVar(&a.Flag.verbose, "verbose", false, "increase verbosity")
	a.FlagSet.BoolVar(&a.Flag.verbose, "v", false, "increase verbosity")
	a.FlagSet.BoolVar(&a.Flag.list, "l", false, "list configured hooks")
	a.FlagSet.BoolVar(&a.Flag.list, "list", false, "list registered hooks")
	a.FlagSet.Usage = c.usage()

	return a.FlagSet.Parse(args)
}

func (a *App) Routes(mux *http.ServeMux) (*http.ServeMux, error) {
	a.Logger.Info("main:hooks", slog.Int("processing hooks", len(a.Cfg.Hooks)))

	mgr := hook.NewManager(a.Name, a.Logger)

	for i := range a.Cfg.Hooks {
		h := a.Cfg.Hooks[i]

		if h.Disabled {
			a.Logger.Debug("main:hooks", "hook", h.Name, "disable", h.Disabled)
			continue
		}

		if err := h.Validate(); err != nil {
			return nil, fmt.Errorf("invalid hook: %q: %w", h.Name, err)
		}

		method := strings.ToUpper(strings.TrimSpace(h.Method))
		pattern := method + " " + h.Endpoint

		switch h.Type {
		case hook.TypeCommand:
			cmd := mgr.NewCommand(&h)
			mux.Handle(pattern, middleware.Auth(a.Cfg.Server.Token, a.DefaultToken, cmd, a.Logger))
		case hook.TypeUpload:
			cmd := mgr.NewUpload(&h)
			mux.Handle(pattern, middleware.Auth(a.Cfg.Server.Token, a.DefaultToken, cmd, a.Logger))
		default:
			a.Logger.Warn("unknown hook type encountered", "type", string(h.Type))
		}
	}

	mux.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	return mux, nil
}

// Init initializes the app by parsing flags and loading config.
func (a *App) Init() error {
	nc := newCommander(a)
	if err := a.parse(nc); err != nil {
		return err
	}

	if a.Flag.verbose {
		parseLogger(a)
	}

	if err := loadConfig(a); err != nil {
		return err
	}

	if a.Flag.list {
		fmt.Fprint(a.Stdout, hook.PrettyHooks(a.Cfg.Hooks))
		cli.ErrAndExit(a.Name, nil)
	}

	if exit, err := a.Dispatch(); exit {
		cli.ErrAndExit(a.Name, err)
	}

	return nil
}

func (a *App) Dispatch() (bool, error) {
	return newCommander(a).dispatch(a.cmd)
}

func (a *App) WriteConfig() error {
	return a.Cfg.write(a.CfgFile)
}

func New(name, version string) *App {
	return &App{
		Name:         name,
		Version:      version,
		Flag:         &arguments{},
		Stdin:        os.Stdin,
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		FlagSet:      flag.NewFlagSet(name, flag.ExitOnError),
		DefaultToken: defaultToken,
	}
}
