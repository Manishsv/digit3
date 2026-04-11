package idgen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"registry-service/internal/config"
)

type Generator interface {
	Generate(ctx context.Context, tenantID string) (string, error)
}

type Client struct {
	baseURL      string
	templateCode string
	orgValue     string
	httpClient   *http.Client
}

type generateRequest struct {
	TemplateCode string            `json:"templateCode"`
	Variables    map[string]string `json:"variables"`
}

type generateResponse struct {
	ID string `json:"id"`
}

func NewClient(cfg config.IDGenConfig) *Client {
	timeout := cfg.HTTPTimeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	return &Client{
		baseURL:      baseURL,
		templateCode: cfg.TemplateCode,
		orgValue:     cfg.OrgValue,
		httpClient:   &http.Client{Timeout: timeout},
	}
}

func (c *Client) Generate(ctx context.Context, tenantID string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("id generator not configured")
	}
	reqBody := generateRequest{
		TemplateCode: c.templateCode,
		Variables: map[string]string{
			"ORG": c.orgValue,
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal idgen request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/generate", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create idgen request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call idgen service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("idgen responded with status %d", resp.StatusCode)
	}

	var genResp generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return "", fmt.Errorf("failed to decode idgen response: %w", err)
	}
	if strings.TrimSpace(genResp.ID) == "" {
		return "", fmt.Errorf("idgen returned empty id")
	}
	return genResp.ID, nil
}

type FallbackGenerator struct{}

func (FallbackGenerator) Generate(ctx context.Context, tenantID string) (string, error) {
	return fmt.Sprintf("%s-%d", strings.ToUpper(tenantID), time.Now().UnixNano()), nil
}
