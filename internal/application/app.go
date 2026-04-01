// Package application...
package application

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/mateconpizza/goairdrop/internal/cli"
	"github.com/mateconpizza/goairdrop/internal/hook"
	"github.com/mateconpizza/goairdrop/internal/server/middleware"
)

type App struct {
	Name    string        // application name
	Version string        // application version
	RepoURL string        // application repository URL
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
	mgr          *hook.Manager
}

type arguments struct {
	args     []string
	verbose  bool
	version  bool
	generate bool
	list     bool
	Webui    bool
	hook     bool
}

type command struct {
	name      string
	nameShort string
	short     string
	value     *bool
}

func (a *App) commands() []command {
	return []command{
		{"hook", "H", "show hook details", &a.Flag.hook},
		{"list", "l", "list hooks", &a.Flag.list},
		{"gen", "g", "generate curl from hook", &a.Flag.generate},
		{"webui", "w", "enable web UI", &a.Flag.Webui},
		{"version", "V", "print version", &a.Flag.version},
		{"verbose", "v", "verbose output", &a.Flag.verbose},
	}
}

func (a *App) parseFlags() error {
	for _, c := range a.commands() {
		a.FlagSet.BoolVar(c.value, c.name, false, c.short)
		a.FlagSet.BoolVar(c.value, c.nameShort, false, c.short)
	}
	a.FlagSet.Usage = a.usage()
	return a.FlagSet.Parse(a.Flag.args)
}

func (a *App) usage() func() {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Usage: %s [flag] [args]\n\n", a.Name)

	cmds := a.commands()
	if len(cmds) > 0 {
		const padding = 18
		sb.WriteString("Flags:\n")
		for _, c := range cmds {
			flagStr := fmt.Sprintf("-%s, --%s", c.nameShort, c.name)
			fmt.Fprintf(&sb, "  %-*s %s\n", padding, flagStr, c.short)
		}
	}

	cfgPath, _ := configPath(a.Name)
	if Exists(cfgPath) {
		sb.WriteString("\nPath:\n")
		fmt.Fprintf(&sb, "  %s\n", cfgPath)
	}

	return func() {
		fmt.Fprint(a.Stdout, sb.String())
	}
}

func (a *App) Routes(mux *http.ServeMux) (*http.ServeMux, error) {
	a.Logger.Info("main:hooks", slog.Int("processing hooks", len(a.mgr.Hooks)))

	for i := range a.mgr.Hooks {
		h := a.mgr.Hooks[i]

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
			cmd := a.mgr.NewCommand(h)
			// FIX: Check in server start if the default token is being use.
			// - remove `middleware.Auth` DefaultToken param
			mux.Handle(pattern, middleware.Auth(a.Cfg.Server.Token, a.DefaultToken, cmd, a.Logger))
		case hook.TypeUpload:
			cmd := a.mgr.NewUpload(h)
			mux.Handle(pattern, middleware.Auth(a.Cfg.Server.Token, a.DefaultToken, cmd, a.Logger))
		default:
			a.Logger.Warn("unknown hook type encountered", "type", string(h.Type))
		}
	}

	mux.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return mux, nil
}

// Init initializes the app by parsing flags and loading config.
func (a *App) Init() error {
	a.Flag.args = os.Args[1:]
	if err := a.parseFlags(); err != nil {
		return err
	}

	if err := loadConfig(a); err != nil {
		return err
	}

	if a.Flag.verbose {
		parseLogger(a)
	}

	if exit, err := a.dispatch(); exit {
		cli.ErrAndExit(a.Name, err)
	}

	return nil
}

func (a *App) dispatch() (bool, error) {
	if a.Flag.version {
		fmt.Fprint(a.Stdout, a.version())
		return true, nil
	}

	if a.Flag.generate {
		return true, a.genCurl(a.Flag.args[1:])
	}

	if a.Flag.hook {
		return true, a.printHook(a.Flag.args[1:])
	}

	if a.Flag.list {
		fmt.Fprint(a.Stdout, a.mgr.PrettifyHooks())
		return true, nil
	}

	return false, nil
}

func (a *App) WriteConfig() error {
	return a.Cfg.write(a.CfgFile)
}

func (a *App) version() string {
	return fmt.Sprintf("%s v%s %s/%s\n", a.Name, a.Version, runtime.GOOS, runtime.GOARCH)
}

func (a *App) genCurl(args []string) error {
	if len(args) == 0 {
		fmt.Fprintf(a.Stdout, "%s: usage: --gen <hook-name> or <hook-endpoint>\n", a.Name)
		return nil
	}

	h, err := a.mgr.Find(args[0])
	if err != nil {
		return err
	}

	baseURL := a.Cfg.Server.Addr
	if strings.HasPrefix(baseURL, ":") {
		baseURL = "localhost" + baseURL
		_ = baseURL
	}

	fmt.Fprintln(a.Stdout, genCurl(h, baseURL))

	return nil
}

func (a *App) printHook(args []string) error {
	if len(args) == 0 {
		fmt.Fprintf(a.Stdout, "%s: usage: --hook <hook-name> or <hook-endpoint>\n", a.Name)
		return nil
	}

	h, err := a.mgr.Find(args[0])
	if err != nil {
		return err
	}

	fmt.Fprintln(a.Stdout, h.String())
	return nil
}

func New(name, version, repo string) *App {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return &App{
		Name:         name,
		Version:      version,
		RepoURL:      repo,
		Flag:         &arguments{},
		Stdin:        os.Stdin,
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
		Logger:       logger,
		FlagSet:      flag.NewFlagSet(name, flag.ExitOnError),
		DefaultToken: defaultToken,
		mgr:          hook.NewManager(name, logger),
	}
}
