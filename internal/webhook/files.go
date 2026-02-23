package webhook

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mateconpizza/goairdrop/internal/cli"
	"github.com/mateconpizza/goairdrop/internal/notify"
)

const DateFormat = "20060102" // DateFormat Date format.

// HandlerFileUploadMultipleNew handles an HTTP POST with one or more uploaded files.
func HandlerFileUploadMultipleNew(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	files, err := parseFiles(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	deviceName := getDeviceName(r)
	uploadDir, err := createUploadDirectory("/tmp/001/goairdrop", deviceName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, fh := range files {
		if err := saveUploadedFile(fh, uploadDir); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

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
		notify.WithAppName("goaird"),
		notify.WithIcon(notify.IconInfo),
		notify.WithUrgency(notify.UrgencyNormal),
		notify.WithID(999),
	)

	if _, err := notification.Send(); err != nil {
		slog.Error("Failed to send notification", slog.String("error", err.Error()))
	}

	_, _ = fmt.Fprintf(w, "Successfully uploaded %d file(s)\n", len(files))
	slog.Info("Successfully uploaded files", slog.Int("count", len(files)))
}

// openFile opens a file in the default file manager.
func openFile(ctx context.Context, p string) error {
	args := cli.OSArgs()
	if err := cli.ExecuteCmd(ctx, append(args, p)...); err != nil {
		return fmt.Errorf("%w: opening in browser", err)
	}

	t := notify.New(
		notify.WithContext(ctx),
		notify.WithBody("Opening file: "+p),
		notify.WithIcon(notify.IconInfo),
		notify.WithUrgency(notify.UrgencyNormal),
		notify.WithID(999),
	)

	if _, err := t.Send(); err != nil {
		slog.Error("Failed to send notification", slog.String("error", err.Error()))
	}

	return nil
}

// parseFiles parses the multipart form and returns the uploaded files.
func parseFiles(r *http.Request) ([]*multipart.FileHeader, error) {
	// Limit the size of the incoming request body (e.g., 10 MB here).
	err := r.ParseMultipartForm(10 << 20) // 10 MB
	if err != nil {
		slog.Error("Error parsing form", slog.String("error", err.Error()))
		return nil, fmt.Errorf("error parsing form: %w", err)
	}

	slog.Info("Form keys", "keys", r.MultipartForm.File)

	// Get all uploaded files from the "file" field
	files := r.MultipartForm.File["file[]"]
	if len(files) == 0 {
		slog.Error("No files uploaded", slog.String("ip", getClientIP(r)))
		return nil, cli.ErrNoFilesUploaded
	}

	return files, nil
}

// saveUploadedFile saves a single uploaded file to the specified directory.
func saveUploadedFile(fh *multipart.FileHeader, directory string) error {
	file, err := fh.Open()
	if err != nil {
		slog.Error("Error opening file", slog.String("error", err.Error()))
		return fmt.Errorf("error opening file %s: %w", fh.Filename, err)
	}

	defer file.Close()

	// Save file to disk
	dst, err := os.Create(filepath.Join(directory, fh.Filename))
	if err != nil {
		slog.Error("Error saving file", slog.String("error", err.Error()))
		return fmt.Errorf("unable to save file %s: %w", fh.Filename, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		slog.Error("Error saving file", slog.String("error", err.Error()))
		return fmt.Errorf("error saving file %s: %w", fh.Filename, err)
	}

	slog.Info("File uploaded", slog.String("filename", fh.Filename), slog.String("path", dst.Name()))

	return nil
}

// createUploadDirectory creates the directory for storing uploaded files.
func createUploadDirectory(root, deviceName string) (string, error) {
	if root == "" {
		return "", cli.ErrPathEmpty
	}
	now := time.Now().Format(DateFormat)
	p := filepath.Join(root, deviceName, now)

	if err := cli.MkdirAll(p); err != nil {
		slog.Error("Error creating directory", slog.String("error", err.Error()))
		return "", fmt.Errorf("error creating directory: %w", err)
	}

	return p, nil
}

func generateFilename(strategy, original string, data []byte) string {
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
