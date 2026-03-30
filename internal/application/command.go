package application

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/mateconpizza/goairdrop/internal/hook"
)

var ErrUnknownCmd = errors.New("unknown command")

const (
	cmdGen     = "gen"
	cmdVersion = "version"
)

type commander struct {
	app *App
	out io.Writer
}

func newCommander(a *App) *commander {
	return &commander{app: a, out: os.Stdout}
}

func (c *commander) dispatch(cmd string) (bool, error) {
	switch cmd {
	case cmdGen:
		return true, c.genCurl(c.app.Flag.args)
	case cmdVersion:
		fmt.Fprint(c.out, c.version())
		return true, nil
	case "":
		return false, nil
	default:
		return true, fmt.Errorf("%w: %s", ErrUnknownCmd, cmd)
	}
}

func (c *commander) version() string {
	return fmt.Sprintf("%s v%s %s/%s\n", c.app.Name, c.app.Version, runtime.GOOS, runtime.GOARCH)
}

func (c *commander) genCurl(args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(c.out, "usage: gen <hook-name> or <hook-endpoint>")
		return nil
	}

	h, err := c.findHook(args[0])
	if err != nil {
		return err
	}
	_ = h

	baseURL := c.app.Cfg.Server.Addr
	if strings.HasPrefix(baseURL, ":") {
		baseURL = "localhost" + baseURL
		_ = baseURL
	}

	fmt.Fprintln(c.out, "not implemented yet")
	return nil
}

func (c *commander) findHook(name string) (*hook.Hook, error) {
	for i := range c.app.Cfg.Hooks {
		h := c.app.Cfg.Hooks[i]
		if h.Name == name || h.Endpoint == name {
			return &h, nil
		}
	}
	return nil, fmt.Errorf("%w: %q", hook.ErrHookNotFound, name)
}

// Usage prints the Usage message.
func (c *commander) usage() func() {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Usage: %s <command> [args]\n\nCommands:\n", c.app.Name)
	for _, cmd := range commands {
		fmt.Fprintf(&sb, "  %-14s %s\n", cmd.name, cmd.short)
	}

	// flags
	sb.WriteString("\nFlags:\n")
	fmt.Fprintf(&sb, "  %-14s %s\n", "-v, --verbose", "verbose mode")
	fmt.Fprintf(&sb, "  %-14s %s\n", "-l, --list", "list registered hooks")

	// add config filepath
	cfgPath, _ := configPath(c.app.Name)
	if Exists(cfgPath) {
		sb.WriteString("\nPath:\n")
		fmt.Fprintf(&sb, "  %s\n", cfgPath)
	}

	return func() {
		fmt.Fprint(c.out, sb.String())
	}
}

var commands = []command{
	{cmdGen, "generate a cURL snippet for a hook"},
	{cmdVersion, "print version and exit"},
}

type command struct {
	name  string
	short string // for usage/help
}

func isKnownCmd(name string) bool {
	for _, c := range commands {
		if c.name == name {
			return true
		}
	}
	return false
}
