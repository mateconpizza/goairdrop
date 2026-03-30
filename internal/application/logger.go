package application

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// newLogger creates a new logger that writes to both the specified file and stdout.
func newLogger(level slog.Level, w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
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
}

func parseLogger(app *App) {
	level := slog.LevelDebug
	writer := os.Stdout
	logger := newLogger(level, writer)
	slog.SetDefault(logger)
	app.Logger = logger
}
