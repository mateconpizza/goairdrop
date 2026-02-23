package application

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mateconpizza/goairdrop/internal/cli"
)

// newLogger creates a new logger that writes to both the specified file and stdout.
func newLogger(filePath string) (*os.File, *slog.Logger, error) {
	const filePerm = 0o644 // Permissions for new files.
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, filePerm)
	if err != nil {
		return nil, nil, fmt.Errorf("%w", err)
	}

	multiWriter := io.MultiWriter(f, os.Stdout)
	logger := slog.New(slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				return slog.String("time", a.Value.Time().Format(time.RFC3339))
			case slog.LevelKey:
				return slog.String("level", strings.ToLower(a.Value.String()))
			case slog.SourceKey:
				if source, ok := a.Value.Any().(*slog.Source); ok {
					dir, file := filepath.Split(source.File)
					shortFile := filepath.Join(filepath.Base(filepath.Clean(dir)), file)
					return slog.String("src", fmt.Sprintf("%s:%d", shortFile, source.Line))
				}
			case slog.MessageKey:
				return a
			}

			return a
		},
	}))

	return f, logger, nil
}

// initDefaultLogger sets up the default logger for the application.
func initDefaultLogger(appName string) (*os.File, *slog.Logger) {
	logFname := logPath(appName)

	f, logger, err := newLogger(logFname)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	slog.SetDefault(logger)

	return f, logger
}

// logPath returns the XDG-compliant log file path for the given name.
func logPath(appName string) string {
	localState := cli.GetEnv("XDG_STATE_HOME", cli.ExpandUser("~/.local/state"))
	return filepath.Join(localState, appName+".json")
}
