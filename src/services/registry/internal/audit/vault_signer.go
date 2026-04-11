package audit

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type VaultConfig struct {
	Address      string
	Token        string
	TransitMount string
	KeyPrefix    string
	Timeout      time.Duration
}

type VaultSigner struct {
	client       *http.Client
	address      string
	token        string
	transitMount string
	keyPrefix    string
}

func NewVaultSigner(cfg VaultConfig) (*VaultSigner, error) {
	if strings.TrimSpace(cfg.Address) == "" {
		return nil, fmt.Errorf("vault address is required")
	}
	if strings.TrimSpace(cfg.Token) == "" {
		return nil, fmt.Errorf("vault token is required")
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 15 * time.Second
	}

	mount := cfg.TransitMount
	if mount == "" {
		mount = "transit"
	}

	prefix := cfg.KeyPrefix
	if prefix == "" {
		prefix = "registry-"
	}

	return &VaultSigner{
		client:       &http.Client{Timeout: timeout},
		address:      strings.TrimRight(cfg.Address, "/"),
		token:        cfg.Token,
		transitMount: strings.Trim(mount, "/"),
		keyPrefix:    prefix,
	}, nil
}

func (v *VaultSigner) Sign(ctx context.Context, tenantID string, digest []byte) (string, int, string, error) {
	keyName := v.keyForTenant(tenantID)
	signature, keyVersion, algo, err := v.signWithKey(ctx, keyName, digest)
	if err == nil {
		return signature, keyVersion, algo, nil
	}

	if !isNotFound(err) {
		return "", 0, "", err
	}

	if err := v.createKey(ctx, keyName); err != nil {
		return "", 0, "", fmt.Errorf("failed to create vault key '%s': %w", keyName, err)
	}

	return v.signWithKey(ctx, keyName, digest)
}

func (v *VaultSigner) Verify(ctx context.Context, tenantID string, digest []byte, signature string) (bool, error) {
	if strings.TrimSpace(signature) == "" {
		return false, fmt.Errorf("signature is required for verification")
	}
	keyName := v.keyForTenant(tenantID)
	return v.verifyWithKey(ctx, keyName, digest, signature)
}

func (v *VaultSigner) signWithKey(ctx context.Context, keyName string, digest []byte) (string, int, string, error) {
	input := base64.StdEncoding.EncodeToString(digest)
	requestBody := map[string]interface{}{
		"input": input,
	}

	url := fmt.Sprintf("%s/v1/%s/sign/%s", v.address, v.transitMount, keyName)
	respBody, status, err := v.performRequest(ctx, http.MethodPost, url, requestBody)
	if err != nil {
		return "", 0, "", err
	}
	if status == http.StatusNotFound {
		return "", 0, "", notFoundError{}
	}
	if status >= 300 {
		return "", 0, "", fmt.Errorf("vault sign request failed with status %d", status)
	}

	var parsed struct {
		Data struct {
			Signature  string `json:"signature"`
			KeyVersion int    `json:"key_version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", 0, "", fmt.Errorf("failed to parse vault sign response: %w", err)
	}
	if parsed.Data.Signature == "" {
		return "", 0, "", fmt.Errorf("vault returned empty signature")
	}

	return parsed.Data.Signature, parsed.Data.KeyVersion, "vault-transit", nil
}

func (v *VaultSigner) verifyWithKey(ctx context.Context, keyName string, digest []byte, signature string) (bool, error) {
	input := base64.StdEncoding.EncodeToString(digest)
	requestBody := map[string]interface{}{
		"input":     input,
		"signature": signature,
	}

	url := fmt.Sprintf("%s/v1/%s/verify/%s", v.address, v.transitMount, keyName)
	respBody, status, err := v.performRequest(ctx, http.MethodPost, url, requestBody)
	if err != nil {
		return false, err
	}
	if status == http.StatusNotFound {
		return false, notFoundError{}
	}
	if status >= 300 {
		return false, fmt.Errorf("vault verify request failed with status %d", status)
	}

	var parsed struct {
		Data struct {
			Valid bool `json:"valid"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return false, fmt.Errorf("failed to parse vault verify response: %w", err)
	}
	return parsed.Data.Valid, nil
}

func (v *VaultSigner) createKey(ctx context.Context, keyName string) error {
	requestBody := map[string]interface{}{
		"type":       "rsa-2048",
		"exportable": true,
	}
	url := fmt.Sprintf("%s/v1/%s/keys/%s", v.address, v.transitMount, keyName)
	_, status, err := v.performRequest(ctx, http.MethodPost, url, requestBody)
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("vault key create failed with status %d", status)
	}
	return nil
}

func (v *VaultSigner) performRequest(ctx context.Context, method, url string, body interface{}) ([]byte, int, error) {
	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal vault request body: %w", err)
		}
	}

	var bodyReader io.Reader
	if reqBody != nil {
		bodyReader = bytes.NewReader(reqBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create vault request: %w", err)
	}
	req.Header.Set("X-Vault-Token", v.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("vault request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read vault response: %w", err)
	}

	return respBytes, resp.StatusCode, nil
}

func (v *VaultSigner) keyForTenant(tenantID string) string {
	normalized := strings.ToLower(tenantID)
	builder := strings.Builder{}
	for _, r := range normalized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
		} else {
			builder.WriteRune('-')
		}
	}
	key := builder.String()
	if key == "" {
		key = "default"
	}
	return fmt.Sprintf("%s%s", v.keyPrefix, key)
}

type notFoundError struct{}

func (notFoundError) Error() string { return "vault key not found" }

func isNotFound(err error) bool {
	_, ok := err.(notFoundError)
	return ok
}
