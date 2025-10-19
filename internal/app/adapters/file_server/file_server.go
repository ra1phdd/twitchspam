package file_server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"twitchspam/pkg/logger"
)

type FileServer struct {
	log    logger.Logger
	client *http.Client
}

func New(log logger.Logger, client *http.Client) *FileServer {
	return &FileServer{
		log:    log,
		client: client,
	}
}

func (fs *FileServer) UploadToHaste(text string) (string, error) {
	fs.log.Debug("Starting file upload to Haste", slog.Int("text_length", len(text)))

	body := strings.NewReader(text)
	req, err := http.NewRequest(http.MethodPost, "https://haste.potat.app/documents", body)
	if err != nil {
		fs.log.Error("Failed to create HTTP request", err)
		return "", err
	}

	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	req.Header.Set("Accept", "application/json")

	resp, err := fs.client.Do(req)
	if err != nil {
		fs.log.Error("HTTP request to Haste failed", err)
		return "", err
	}
	defer resp.Body.Close()

	fs.log.Debug("Received response from Haste", slog.Int("status_code", resp.StatusCode))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		fs.log.Error("Unexpected HTTP status during upload", nil,
			slog.Int("status_code", resp.StatusCode),
			slog.String("status", resp.Status),
			slog.String("raw", string(raw)),
		)
		return "", fmt.Errorf("unexpected status %s", resp.Status)
	}

	var out struct {
		Key string `json:"key"`
	}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&out); err != nil {
		fs.log.Error("Failed to decode Haste response", err)
		return "", err
	}

	if out.Key == "" {
		fs.log.Error("Empty key received in Haste response", nil)
		return "", errors.New("empty key in response")
	}

	fs.log.Info("File successfully uploaded to Haste", slog.String("key", out.Key))
	return out.Key, nil
}

func (fs *FileServer) GetURL(key string) string {
	url := "https://haste.potat.app/raw/" + key
	fs.log.Trace("Generated Haste file URL", slog.String("url", url))
	return url
}
