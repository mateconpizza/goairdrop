package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

var (
	ErrFileExists = errors.New("file already exists")
	ErrPathEmpty  = errors.New("path is empty")
)

const (
	DirPerm  = 0o755 // Permissions for new directories.
	FilePerm = 0o644 // Permissions for new files.
)

// writeJSON writes the provided data as JSON to the specified file.
func writeJSON[T any](p string, v *T) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	if err := os.WriteFile(p, data, FilePerm); err != nil {
		return fmt.Errorf("write file %q: %w", p, err)
	}

	slog.Debug("json write success", "path", p)
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

// mkdir creates a new directory at the specified path.
func mkdir(s string) error {
	if Exists(s) {
		return nil
	}

	slog.Debug("creating path", "path", s)

	if err := os.MkdirAll(s, DirPerm); err != nil {
		return fmt.Errorf("creating %s: %w", s, err)
	}

	return nil
}

// Exists checks if a file exists.
func Exists(s string) bool {
	_, err := os.Stat(s)
	return !os.IsNotExist(err)
}

// Touch creates a file at this given path.
// If the file already exists, the function succeeds when exist_ok is true.
func Touch(s string, existsOK bool) (*os.File, error) {
	if Exists(s) && !existsOK {
		return nil, fmt.Errorf("%w: %q", ErrFileExists, s)
	}

	if !Exists(filepath.Dir(s)) {
		if err := MkdirAll(filepath.Dir(s)); err != nil {
			return nil, err
		}
	}

	f, err := os.Create(s)
	if err != nil {
		return nil, fmt.Errorf("error creating file: %w", err)
	}

	return f, nil
}
