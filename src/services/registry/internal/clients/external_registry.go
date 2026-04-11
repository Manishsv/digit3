package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"registry-service/internal/config"
)

type ExternalRegistryResolver interface {
	Exists(ctx context.Context, registryName, schemaCode, tenantID, field, value string) (bool, error)
}

type externalRegistryClient struct {
	configs map[string]config.ExternalRegistryConfig
}

func NewExternalRegistryClient(cfg map[string]config.ExternalRegistryConfig) ExternalRegistryResolver {
	if len(cfg) == 0 {
		return nil
	}
	return &externalRegistryClient{configs: cfg}
}

func (c *externalRegistryClient) Exists(ctx context.Context, registryName, schemaCode, tenantID, field, value string) (bool, error) {
	if c == nil {
		return false, fmt.Errorf("external registry resolver is not configured")
	}
	cfg, ok := c.configs[registryName]
	if !ok {
		return false, fmt.Errorf("registry '%s' not configured", registryName)
	}
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		return false, fmt.Errorf("registry '%s' base url not configured", registryName)
	}

	endpoint := fmt.Sprintf("%s/schema/%s/_isExist", baseURL, url.PathEscape(schemaCode))

	payload, err := json.Marshal(map[string]string{
		"tenantId": tenantID,
		"field":    field,
		"value":    value,
	})
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID)
	for k, v := range cfg.Headers {
		if strings.TrimSpace(k) == "" {
			continue
		}
		req.Header.Set(k, v)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	client := &http.Client{Timeout: timeout}

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("registry '%s' responded with status %d", registryName, resp.StatusCode)
	}

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Exists bool `json:"exists"`
		} `json:"data"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode response from registry '%s': %w", registryName, err)
	}

	if !result.Success {
		return false, fmt.Errorf("registry '%s' returned error: %s", registryName, result.Error)
	}

	return result.Data.Exists, nil
}
