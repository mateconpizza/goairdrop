package hook

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mateconpizza/goairdrop/internal/cli"
	"github.com/mateconpizza/goairdrop/internal/notify"
)

var ErrInvalidFilename = errors.New("invalid filename")

func (m *Manager) NewUpload(h *Hook) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		files, err := parseFiles(r, m.logger)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		deviceName := getDeviceName(r)
		uploadDir := cli.ExpandUser(h.Destination)

		for _, fh := range files {
			if err := saveUploadedFile(fh, uploadDir, m.logger); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		if h.Notify {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			notification := notify.New(
				notify.WithContext(ctx),
				notify.WithBody(
					fmt.Sprintf(
						"Received %d file(s) from device: %s\n",
						len(files),
						deviceName+" ("+getClientIP(r)+")",
					),
				),
				notify.WithAppName(m.appName),
				notify.WithIcon(notify.IconInfo),
				notify.WithUrgency(notify.UrgencyNormal),
				notify.WithID(999),
			)

			if _, err := notification.Send(); err != nil {
				m.logger.Error("Failed to send notification", "error", err.Error())
			}
		}

		resp := Response{
			Success: true,
			Message: fmt.Sprintf("Successfully uploaded %d file(s)", len(files)),
		}

		m.logger.Info(resp.Message)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			m.logger.Error("Error encoding response", "error", err.Error())
		}
	}
}

// parseFiles parses the multipart form and returns the uploaded files.
func parseFiles(r *http.Request, logger *slog.Logger) ([]*multipart.FileHeader, error) {
	// Limit the size of the incoming request body (e.g., 10 MB here).
	err := r.ParseMultipartForm(10 << 20) // 10 MB
	if err != nil {
		logger.Error("Error parsing form", "error", err.Error())
		return nil, fmt.Errorf("error parsing form: %w", err)
	}

	logger.Info("Form keys", "keys", r.MultipartForm.File)

	// Get all uploaded files from the "file" field
	// FIX: use `FormField`
	files := r.MultipartForm.File["file[]"]
	if len(files) == 0 {
		logger.Error("No files uploaded", slog.String("ip", getClientIP(r)))
		return nil, cli.ErrNoFilesUploaded
	}

	return files, nil
}

// sanitizeUploadPath validates and sanitizes a filename from an untrusted source,
// returning the safe absolute path within the given directory.
func sanitizeUploadPath(directory, filename string) (string, error) {
	cleanName := filepath.Base(filepath.FromSlash(filename))

	if cleanName == "" || cleanName == "." {
		return "", fmt.Errorf("%w: %q", ErrInvalidFilename, filename)
	}

	absDir, err := filepath.Abs(directory)
	if err != nil {
		return "", fmt.Errorf("invalid upload directory: %w", err)
	}

	absTarget, err := filepath.Abs(filepath.Join(absDir, cleanName))
	if err != nil {
		return "", fmt.Errorf("invalid target path: %w", err)
	}

	if !strings.HasPrefix(absTarget, absDir+string(os.PathSeparator)) {
		return "", fmt.Errorf("%w: %q: resolves outside upload directory", ErrInvalidFilename, filename)
	}

	return absTarget, nil
}

func saveUploadedFile(fh *multipart.FileHeader, directory string, logger *slog.Logger) error {
	targetPath, err := sanitizeUploadPath(directory, fh.Filename)
	if err != nil {
		return fmt.Errorf("unsafe upload filename: %w", err)
	}

	file, err := fh.Open()
	if err != nil {
		logger.Error("Error opening file", "error", err)
		return fmt.Errorf("error opening file %s: %w", fh.Filename, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Error("Error closing file", "error", err)
		}
	}()

	dst, err := os.Create(targetPath)
	if err != nil {
		logger.Error("Error saving file", "error", err)
		return fmt.Errorf("unable to save file %s: %w", targetPath, err)
	}
	defer func() {
		if err := dst.Close(); err != nil {
			logger.Error("Error closing destination file", "error", err)
		}
	}()

	if _, err := io.Copy(dst, file); err != nil {
		logger.Error("Error saving file", "error", err)
		return fmt.Errorf("error saving file %s: %w", targetPath, err)
	}
	logger.Info("File uploaded", "filename", filepath.Base(targetPath), "path", targetPath)
	return nil
}

func generateFilename(strategy, original string, data []byte) string {
	// FIX: Finish implementation
	ext := filepath.Ext(original)

	switch strategy {
	// case "uuid":
	// 	return uuid.New().String() + ext

	case "timestamp":
		return time.Now().UTC().Format("20060102T150405") + ext

	case "hash":
		sum := sha256.Sum256(data)
		return hex.EncodeToString(sum[:]) + ext

	case "original":
		return sanitize(original)

	default:
		panic("invalid filename strategy")
	}
}

func sanitize(s string) string {
	// TODO: finish it.
	return s
}
