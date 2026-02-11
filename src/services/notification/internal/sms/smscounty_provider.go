package sms

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"notification/internal/config"
	"notification/internal/models"
	"strings"
)

// SMSCountryProvider is an implementation of SMSProvider for SMSCountry.
type SMSCountryProvider struct {
	config *config.Config
	client *http.Client
}

// NewSMSCountryProvider creates a new SMSCountryProvider.
func NewSMSCountryProvider(config *config.Config) *SMSCountryProvider {
	return &SMSCountryProvider{
		config: config,
		client: &http.Client{},
	}
}

// Send sends an SMS using the SMSCountry provider.
func (p *SMSCountryProvider) Send(mobileNumber, message string) []models.Error {
	data := url.Values{}
	data.Set("User", p.config.SMSProviderUsername)
	data.Set("passwd", p.config.SMSProviderPassword)
	data.Set("mobilenumber", mobileNumber)
	data.Set("message", message)
	data.Set("mtype", "N")

	req, err := http.NewRequest("POST", p.config.SMSProviderURL, strings.NewReader(data.Encode()))
	if err != nil {
		return []models.Error{{
			Code:        "CREATE__REQUEST_FAILED",
			Message:     "Failed to create request",
			Description: err.Error(),
		}}
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return []models.Error{{
			Code:        "SEND_SMS_FAILED",
			Message:     "Failed to send SMS",
			Description: err.Error(),
		}}
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return []models.Error{{
			Code:        "READ_RESPONSE_FAILED",
			Message:     "Failed to read SMS response",
			Description: readErr.Error(),
		}}
	}

	// Log the response body for visibility
	fmt.Printf("SMSCountry response: %s\n", string(body))

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return []models.Error{{
			Code:        "SEND_SMS_FAILED",
			Message:     "Failed to send SMS",
			Description: fmt.Sprintf("Status code: %d, Response: %s", resp.StatusCode, string(body)),
		}}
	}

	return nil
}
