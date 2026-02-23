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

	"github.com/mateconpizza/goairdrop/internal/webhook"
)

const FilePerm = 0o644 // Permissions for new files.

type App struct {
	Name    string        // application name
	Version string        // application version
	CmdArgs *Args         // parsed command-line arguments
	FlagSet *flag.FlagSet // command-line flag parser

	Cfg     *Config // application config
	CfgFile string  // config filepath

	Stdin  io.Reader // standard input
	Stdout io.Writer // standard output
	Stderr io.Writer // standard error

	Logger  *slog.Logger // application logger
	LogFile *os.File     // log file
}

type Args struct {
	Addr    string
	Version bool
}

func (a *App) Ver() string {
	return fmt.Sprintf("%s v%s %s/%s\n", a.Name, a.Version, runtime.GOOS, runtime.GOARCH)
}

func (a *App) Parse() error {
	return a.parseFlags(os.Args[1:])
}

func (a *App) parseFlags(args []string) error {
	a.FlagSet.StringVar(&a.CmdArgs.Addr, "addr", ":5001", "HTTP service address")

	a.FlagSet.BoolVar(&a.CmdArgs.Version, "version", false, "Print version and exit")
	a.FlagSet.BoolVar(&a.CmdArgs.Version, "V", false, "Print version and exit")

	a.FlagSet.Usage = a.usage(logPath(a.Name))

	return a.FlagSet.Parse(args)
}

// Usage prints the Usage message.
func (a *App) usage(fn string) func() {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Usage:  %s v%s [options]\n\n", a.Name, a.Version)
	sb.WriteString("\tSimple webhook server\n\n")
	sb.WriteString("Options:\n")
	sb.WriteString("  -a, -addr string\n\tHTTP service address (default \":5001\")\n")
	sb.WriteString("  -V, -version\n\tPrint version and exit\n")
	sb.WriteString("  -b, -beta\n\tbeta test\n")
	sb.WriteString("\nFiles:\n")
	fmt.Fprintf(&sb, "\t%s\n", fn)

	return func() {
		fmt.Fprint(os.Stderr, sb.String())
	}
}

func (a *App) Error(err error) {
	slog.Error(a.Name, slog.String("error", err.Error()))
	_, _ = fmt.Fprintf(a.Stderr, "%s: %s\n", a.Name, err)
}

func (a *App) LoadConfig() error {
	path, err := configPath(a.Name)
	if err != nil {
		return err
	}

	cfg, err := parse(path)
	if err != nil {
		return err
	}

	a.Cfg = cfg

	return nil
}

func (a *App) SetupRoutes() (*http.ServeMux, error) {
	mux := http.NewServeMux()

	// temporary
	// mux.HandleFunc("/debug-test", func(w http.ResponseWriter, r *http.Request) {
	// 	_, _ = w.Write([]byte("ok"))
	// })

	slog.Info("main:hooks", slog.Int("processing hooks", len(a.Cfg.Hooks)))

	for i := range a.Cfg.Hooks {
		hook := a.Cfg.Hooks[i]

		if err := hook.Validate(); err != nil {
			return nil, fmt.Errorf("validate hook '%s': %w", hook.Name, err)
		}

		switch hook.Type {
		case webhook.TypeCommand:
			mux.HandleFunc(hook.Endpoint, webhook.HandleCommandHook(&hook))
		case webhook.TypeUpload:
			mux.HandleFunc(hook.Endpoint, webhook.HandleUploadHook(&hook))
		default:
			slog.Warn("Unknown hook type encountered", slog.String("type", string(hook.Type)))
		}
	}

	return mux, nil
}

func (a *App) Run() error {
	if err := a.LoadConfig(); err != nil {
		return err
	}

	mux, err := a.SetupRoutes()
	if err != nil {
		return err
	}

	server := webhook.New(a.CmdArgs.Addr, mux)

	slog.Info("Server starting on " + a.CmdArgs.Addr)

	return server.Start()
}

func New(name, version string) *App {
	logFile, logger := initDefaultLogger(name)

	return &App{
		Name:    name,
		Version: version,
		CmdArgs: &Args{},
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Logger:  logger,
		LogFile: logFile,
		FlagSet: flag.NewFlagSet(name, flag.ExitOnError),
	}
}
