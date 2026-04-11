package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"registry-service/internal/models"
	"time"
)

type CallbackManager struct {
	client *http.Client
}

func NewCallbackManager() *CallbackManager {
	return &CallbackManager{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (cm *CallbackManager) ExecuteCallback(config models.CallbackConfig, payload interface{}) error {
	if config.URL == "" {
		return nil // No callback configured
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal callback payload: %w", err)
	}

	req, err := http.NewRequest(config.Method, config.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create callback request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add custom headers
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	resp, err := cm.client.Do(req)
	if err != nil {
		return fmt.Errorf("callback request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("callback returned error status: %d", resp.StatusCode)
	}

	log.Printf("Callback executed successfully: %s %s", config.Method, config.URL)
	return nil
}