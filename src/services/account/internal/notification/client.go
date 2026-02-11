package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Client struct {
	BaseURL string
}

type NotificationTemplate struct {
	TemplateID string `json:"templateId"`
	Version    string `json:"version"`
	Type       string `json:"type"`
	Subject    string `json:"subject"`
	Content    string `json:"content"`
	IsHTML     bool   `json:"isHTML"`
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

func (c *Client) CreateTemplate(tenantCode string, template *NotificationTemplate) error {
	url := fmt.Sprintf("%s/notification/template", c.BaseURL)
	
	jsonData, err := json.Marshal(template)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", strings.ToUpper(tenantCode))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("notification template creation failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) CreateSandboxEmailTemplate(tenantCode, baseURL, tenantID string) error {
	template := &NotificationTemplate{
		TemplateID: "sandbox-email-login-otp",
		Version:    "1.0.0",
		Type:       "EMAIL",
		Subject:    "Sandbox Sign-up/Login OTP",
		Content: fmt.Sprintf(`Dear User,<br><br>To complete sign-up or login to your Sandbox Account, please enter the below OTP:<br><br><b style='font-size: 24px; color: #000;'>{{.otp}}</b><br><br>Your exclusive login URL is <a href='%ssandbox-ui/%s/employee'>%ssandbox-ui/%s/employee</a><br><br>Please bookmark and use this URL for future access to Sandbox.<br><br>If you did not initiate this action, please contact <a href='mailto:digit.sandbox@egovernments.org'>digit.sandbox@egovernments.org</a><br><br>Regards,<br>Sandbox Team`, 
			baseURL, tenantID, baseURL, tenantID),
		IsHTML: true,
	}

	return c.CreateTemplate(tenantCode, template)
}
