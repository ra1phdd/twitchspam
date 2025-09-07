package file_server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type FileServer struct{}

func New() *FileServer {
	return &FileServer{}
}

func (fs *FileServer) UploadToHaste(text string) (string, error) {
	body := strings.NewReader(text)

	req, err := http.NewRequest(http.MethodPost, "https://haste.potat.app/documents", body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slurp, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("unexpected status %s: %s", resp.Status, string(slurp))
	}

	var out struct {
		Key string `json:"key"`
	}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&out); err != nil {
		return "", err
	}
	if out.Key == "" {
		return "", errors.New("empty key in response")
	}
	return out.Key, nil
}

func (fs *FileServer) GetURL(key string) string {
	return "https://haste.potat.app/raw/" + key
}
