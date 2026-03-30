// Package cli...
package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var (
	ErrNoFilesUploaded = errors.New("no files uploaded")
	ErrPathEmpty       = errors.New("path is empty")
)

const (
	DirectoryPerm = 0o755 // Permissions for new directories.
	FilePerm      = 0o644 // Permissions for new files.
)

const (
	red   = "\x1b[31m"
	green = "\x1b[32m"
	reset = "\x1b[0m"
)

func Red(s string) string {
	return red + s + reset
}

func Green(s string) string {
	return green + s + reset
}

// AnsiRemover removes ANSI codes from a given string.
func AnsiRemover(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(s, "")
}

// Exit codes used by the application.
const (
	// ExitSuccess indicates normal termination.
	ExitSuccess = 0

	// ExitInterrupted is the conventional exit code for Ctrl+C (SIGINT).
	ExitInterrupted = 130

	// ExitFailure indicates a general failure or unhandled error.
	ExitFailure = 1
)

// OSArgs returns the correct arguments for the OS.
func OSArgs() []string {
	// FIX: only support linux
	var args []string
	switch runtime.GOOS {
	case "darwin":
		args = append(args, "open")
	case "windows":
		args = append(args, "cmd", "/C", "start")
	default:
		args = append(args, "xdg-open")
	}

	return args
}

// ExecuteCmd runs a command with the given arguments in the background and returns
// immediately without waiting for the command to complete.
func ExecuteCmd(ctx context.Context, arg ...string) error {
	cmd := exec.CommandContext(ctx, arg[0], arg[1:]...)

	// Start the command without waiting for it to complete
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting command: %w", err)
	}

	// Release resources associated with the process immediately
	go func() {
		_ = cmd.Wait()
	}()

	return nil
}

// Exists checks if a file Exists.
func Exists(s string) bool {
	_, err := os.Stat(s)
	return !os.IsNotExist(err)
}

// mkdir creates a new directory at the specified path.
func mkdir(s string) error {
	if Exists(s) {
		return nil
	}

	slog.Debug("creating path", "path", s)

	if err := os.MkdirAll(s, DirectoryPerm); err != nil {
		return fmt.Errorf("creating %s: %w", s, err)
	}

	return nil
}

// MkdirAll creates all the given paths.
func MkdirAll(s ...string) error {
	for _, p := range s {
		if p == "" {
			return ErrPathEmpty
		}

		if err := mkdir(p); err != nil {
			return err
		}
	}

	return nil
}

// GetEnv retrieves an environment variable.
//
// If the environment variable is not set, returns the default value.
func GetEnv(s, def string) string {
	if v, ok := os.LookupEnv(s); ok {
		return v
	}

	return def
}

// ExpandUser expands the home directory in the given string.
func ExpandUser(s string) string {
	if strings.HasPrefix(s, "~/") {
		dirname, _ := os.UserHomeDir()
		s = filepath.Join(dirname, s[2:])
	}

	return s
}

// ErrAndExit logs the error and exits the program.
func ErrAndExit(appName string, err error) {
	if err == nil {
		os.Exit(ExitSuccess)
	}

	fmt.Fprintf(os.Stderr, "%s: %s\n", appName, err)
	os.Exit(ExitFailure)
}

func XDGDataHome(appName string) string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, appName, "uploads")
}

// Table generates a simple ASCII table with basic borders.
func Table(headers []string, rows [][]string, footer ...string) string {
	// FIX: refactor and use builder pattern???
	if len(headers) == 0 {
		return ""
	}

	// Compute column widths ignoring ANSI sequences
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = len(AnsiRemover(header))
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) {
				w := len(AnsiRemover(cell))
				if w > colWidths[i] {
					colWidths[i] = w
				}
			}
		}
	}

	var b strings.Builder

	writeBorder := func() {
		b.WriteString("+")
		for _, width := range colWidths {
			b.WriteString(strings.Repeat("-", width+2) + "+")
		}
		b.WriteString("\n")
	}

	writeBorder()

	// Header
	b.WriteString("|")
	for i, header := range headers {
		visibleLen := len(AnsiRemover(header))
		padding := colWidths[i] - visibleLen
		b.WriteString(" " + header + strings.Repeat(" ", padding) + " |")
	}
	b.WriteString("\n")

	writeBorder()

	// Rows
	for _, row := range rows {
		b.WriteString("|")
		for i, width := range colWidths {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			visibleLen := len(AnsiRemover(cell))
			padding := width - visibleLen
			b.WriteString(" " + cell + strings.Repeat(" ", padding) + " |")
		}
		b.WriteString("\n")
	}

	writeBorder()

	// Footer (centered, no borders)
	if len(footer) > 0 {
		totalWidth := 1 // start with first "+"
		for _, w := range colWidths {
			totalWidth += w + 3 // "-" * width + 2 padding + "+"
		}

		totalWidth-- // remove last "+"
		for _, line := range footer {
			lineStripped := AnsiRemover(line)
			lineLen := min(len(lineStripped), totalWidth)
			leftPad := (totalWidth - lineLen) / 2
			b.WriteString(strings.Repeat(" ", leftPad))
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	return b.String()
}
