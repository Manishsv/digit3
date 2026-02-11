package utils

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/go-resty/resty/v2"
)

// RenderRequest represents the request to the template-config service
type RenderRequest struct {
	TemplateID string                 `json:"templateId"`
	TenantID   string                 `json:"tenantId"`
	Version    string                 `json:"version"`
	Payload    map[string]interface{} `json:"payload"`
}

// RenderResponse represents the response from the template-config service
type RenderResponse struct {
	TemplateID string                 `json:"templateId"`
	TenantID   string                 `json:"tenantId"`
	Version    string                 `json:"version"`
	Data       map[string]interface{} `json:"data"`
}

// EnrichmentClient handles communication with the template-config service
type EnrichmentClient struct {
	client *resty.Client
	host   string
	path   string
}

// NewEnrichmentClient creates a new enrichment client
func NewEnrichmentClient(host, path string) *EnrichmentClient {
	client := resty.New()
	client.SetHeader("Content-Type", "application/json")

	return &EnrichmentClient{
		client: client,
		host:   host,
		path:   path,
	}
}

// EnrichPayload enriches the payload using the template-config service
func (ec *EnrichmentClient) EnrichPayload(ctx context.Context, templateID, tenantID, version string, payload map[string]interface{}) (map[string]interface{}, error) {
	request := RenderRequest{
		TemplateID: templateID,
		TenantID:   tenantID,
		Version:    version,
		Payload:    payload,
	}

	var response RenderResponse
	resp, err := ec.client.R().
		SetContext(ctx).
		SetHeader("X-Tenant-ID", tenantID).
		SetBody(request).
		SetResult(&response).
		Post(ec.host + ec.path)

	if err != nil {
		log.Printf("Template-config render request failed: %v", err)
		return nil, fmt.Errorf("failed to call template-config service: %v", err)
	}

	if resp.StatusCode() != http.StatusOK {
		log.Printf("Template-config service returned status %d: %s", resp.StatusCode(), resp.String())
		return nil, fmt.Errorf("template-config service error: status %d", resp.StatusCode())
	}

	return response.Data, nil
}
