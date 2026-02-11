package utils

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-resty/resty/v2"
)

type FileData struct {
	Filename string
	Path     string
}

type FilestoreClient struct {
	client *resty.Client
	host   string
	path   string
}

func NewFilestoreClient(host, path string) *FilestoreClient {
	client := resty.New()
	client.SetHeader("Content-Type", "application/json")

	return &FilestoreClient{
		client: client,
		host:   host,
		path:   path,
	}
}

func (fc *FilestoreClient) GetFile(ctx context.Context, fileID, tenantID string) (*FileData, error) {
	url := fmt.Sprintf("%s%s/%s?tenantId=%s", fc.host, fc.path, fileID, tenantID)

	resp, err := fc.client.R().
		SetContext(ctx).
		SetDoNotParseResponse(true). // Prevent Resty from buffering large files
		Get(url)

	if err != nil {
		return nil, fmt.Errorf("failed to get file from filestore: %w", err)
	}
	defer resp.RawBody().Close()

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("filestore returned status %d", resp.StatusCode())
	}

	// Parse filename from header
	filename := fileID // fallback
	if cd := resp.Header().Get("Content-Disposition"); cd != "" {
		_, params, err := mime.ParseMediaType(cd)
		if err == nil {
			if fname, ok := params["filename"]; ok {
				filename = fname
			}
		}
	}

	// Write to temp file
	tmpDir := os.TempDir()
	tmpFilePath := filepath.Join(tmpDir, fmt.Sprintf("%s_attachment", fileID))

	out, err := os.Create(tmpFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.RawBody()); err != nil {
		return nil, fmt.Errorf("failed to write file to disk: %w", err)
	}

	return &FileData{
		Path:     tmpFilePath,
		Filename: filename,
	}, nil
}
