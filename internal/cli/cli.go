package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
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

// findBinPath finds a binary in $PATH.
func findBinPath(s string) string {
	path := os.Getenv("PATH")
	pathList := filepath.SplitList(path)
	var command string

	for _, directory := range pathList {
		fullPath := filepath.Join(directory, s)
		// Does it exist?
		fileInfo, err := os.Stat(fullPath)
		if err != nil {
			continue
		}

		mode := fileInfo.Mode()
		// Is it a regular file?
		if mode.IsRegular() && mode&0o111 != 0 {
			command = fullPath
			break
		}
	}

	slog.Debug("which command", "cmd", command)

	return command
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
