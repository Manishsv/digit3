package keycloak

import (
	"account/internal/config"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type KeycloakClient struct {
	BaseURL   string
	AdminUser string
	AdminPass string
	Config    *config.KeycloakConfig
}

type RealmConfigData struct {
	TenantCode             string
	TenantCodeLowerCase    string
	TenantEmail            string
	TenantName             string
	MobileNumber           string
	AuthBaseUrl            string
	AuthAdminUrl           string
	AuthServerClientSecret string // Added for template
}

func NewKeycloakClient(baseURL, adminUser, adminPass string, cfg *config.KeycloakConfig) *KeycloakClient {
	return &KeycloakClient{
		BaseURL:   baseURL,
		AdminUser: adminUser,
		AdminPass: adminPass,
		Config:    cfg,
	}
}

func (kc *KeycloakClient) GetAdminToken() (string, error) {
	url := fmt.Sprintf("%s/realms/master/protocol/openid-connect/token", kc.BaseURL)
	data := []byte(fmt.Sprintf("username=%s&password=%s&grant_type=password&client_id=admin-cli", kc.AdminUser, kc.AdminPass))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	if token, ok := result["access_token"].(string); ok {
		return token, nil
	}
	return "", fmt.Errorf("failed to get access token: %s", string(body))
}

func (kc *KeycloakClient) CreateRealmWithFullConfig(tenantCode, tenantEmail, tenantName, mobileNumber string) error {
	// Get admin token
	token, err := kc.GetAdminToken()
	if err != nil {
		return fmt.Errorf("failed to get admin token: %v", err)
	}

	// Load and process realm configuration template
	configData := RealmConfigData{
		TenantCode:             tenantCode,
		TenantCodeLowerCase:    strings.ToLower(tenantCode),
		TenantEmail:            tenantEmail,
		TenantName:             tenantName,
		MobileNumber:           mobileNumber,
		AuthBaseUrl:            kc.BaseURL,
		AuthAdminUrl:           kc.BaseURL,
		AuthServerClientSecret: "changeme", // Set your actual secret here
	}

	realmConfig, err := kc.loadAndProcessRealmConfig(configData)
	if err != nil {
		return fmt.Errorf("failed to load realm config: %v", err)
	}

	// Create realm with full configuration
	if err := kc.createRealmWithConfig(realmConfig, token); err != nil {
		return fmt.Errorf("failed to create realm: %v", err)
	}

	// Set up cross-realm token exchange with CITIZEN realm
	if err := kc.setupCitizenTokenExchange(tenantCode, token); err != nil {
		return fmt.Errorf("failed to setup citizen token exchange: %v", err)
	}

	return nil
}

func (kc *KeycloakClient) loadAndProcessRealmConfig(data RealmConfigData) (map[string]interface{}, error) {
	// Try to read the realm config file using robust path resolution
	templateBytes, err := kc.findAndReadRealmConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read realm config file: %v", err)
	}

	// Parse template
	tmpl, err := template.New("realmConfig").Parse(string(templateBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %v", err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %v", err)
	}

	// Parse JSON
	var config map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return config, nil
}

// findAndReadRealmConfig tries multiple possible paths to find the realm config file
func (kc *KeycloakClient) findAndReadRealmConfig() ([]byte, error) {
	// Get the directory of the current executable
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %v", err)
	}
	execDir := filepath.Dir(execPath)

	// List of possible paths to try
	possiblePaths := []string{}

	// Add configured path first if it's not empty
	if kc.Config.RealmConfigPath != "" {
		possiblePaths = append(possiblePaths, kc.Config.RealmConfigPath)
	}

	// Add fallback paths
	possiblePaths = append(possiblePaths, []string{
		"internal/keycloak/realm_config.json",                               // Relative to current working directory
		filepath.Join(execDir, "internal/keycloak/realm_config.json"),       // Relative to executable
		filepath.Join(execDir, "realm_config.json"),                         // In executable directory
		"realm_config.json",                                                 // Current working directory
		filepath.Join(execDir, "..", "internal/keycloak/realm_config.json"), // One level up from executable
	}...)

	// Try each path
	for _, path := range possiblePaths {
		if data, err := os.ReadFile(path); err == nil {
			return data, nil
		}
	}

	return nil, fmt.Errorf("realm config file not found in any of the expected locations: %v", possiblePaths)
}

func (kc *KeycloakClient) createRealmWithConfig(config map[string]interface{}, token string) error {
	url := fmt.Sprintf("%s/admin/realms", kc.BaseURL)
	jsonBody, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 && resp.StatusCode != 409 { // 409 if already exists
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to create realm, status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// setupCitizenTokenExchange configures cross-realm token exchange with CITIZEN realm
func (kc *KeycloakClient) setupCitizenTokenExchange(tenantCode, token string) error {
	// Step 1: Create OIDC Identity Provider
	if err := kc.createCitizenIdentityProvider(tenantCode, token); err != nil {
		return fmt.Errorf("failed to create citizen identity provider: %v", err)
	}

	// Step 2: Enable token exchange permissions on the identity provider
	if err := kc.enableIdpTokenExchange(tenantCode, token); err != nil {
		return fmt.Errorf("failed to enable token exchange permissions: %v", err)
	}

	// Step 3: Create and attach client policy for auth-server client
	if err := kc.attachAuthServerClientPolicy(tenantCode, token); err != nil {
		return fmt.Errorf("failed to attach auth-server client policy: %v", err)
	}

	return nil
}

// createCitizenIdentityProvider creates the OIDC Identity Provider for CITIZEN realm
func (kc *KeycloakClient) createCitizenIdentityProvider(tenantCode, token string) error {
	url := fmt.Sprintf("%s/admin/realms/%s/identity-provider/instances", kc.BaseURL, tenantCode)

	idpConfig := map[string]interface{}{
		"alias":                       "citizen",
		"displayName":                 "Citizen Identity Provider",
		"providerId":                  "oidc",
		"enabled":                     true,
		"updateProfileFirstLoginMode": "on",
		"trustEmail":                  false,
		"storeToken":                  false,
		"addReadTokenRoleOnCreate":    false,
		"authenticateByDefault":       false,
		"linkOnly":                    false,
		"firstBrokerLoginFlowAlias":   "first broker login",
		"config": map[string]interface{}{
			"clientId":          kc.Config.CitizenBrokerClientId,
			"clientSecret":      kc.Config.CitizenBrokerClientSecret,
			"discoveryEndpoint": fmt.Sprintf("%s/realms/CITIZEN/.well-known/openid-configuration", kc.BaseURL),
			"useJwksUrl":        "true",
			"syncMode":          "IMPORT",
			"authorizationUrl":  fmt.Sprintf("%s/realms/CITIZEN/protocol/openid-connect/auth", kc.BaseURL),
			"tokenUrl":          fmt.Sprintf("%s/realms/CITIZEN/protocol/openid-connect/token", kc.BaseURL),
			"userInfoUrl":       fmt.Sprintf("%s/realms/CITIZEN/protocol/openid-connect/userinfo", kc.BaseURL),
			"issuer":            fmt.Sprintf("%s/realms/CITIZEN", kc.BaseURL),
			"jwksUrl":           fmt.Sprintf("%s/realms/CITIZEN/protocol/openid-connect/certs", kc.BaseURL),
			"validateSignature": "true",
			"clientAuthMethod":  "client_secret_post",
			"pkceEnabled":       "false",
			"defaultScope":      "openid profile email",
		},
	}

	return kc.makeKeycloakRequest("POST", url, idpConfig, token, "create identity provider")
}

// enableIdpTokenExchange enables token exchange permissions on the citizen identity provider
func (kc *KeycloakClient) enableIdpTokenExchange(tenantCode, token string) error {
	// Enable permissions management on the citizen identity provider
	permissionUrl := fmt.Sprintf("%s/admin/realms/%s/identity-provider/instances/citizen/management/permissions", kc.BaseURL, tenantCode)

	permissionConfig := map[string]interface{}{
		"enabled": true,
	}

	return kc.makeKeycloakRequest("PUT", permissionUrl, permissionConfig, token, "enable identity provider permissions")
}

// attachAuthServerClientPolicy creates and attaches client policy for auth-server client to token exchange permission
func (kc *KeycloakClient) attachAuthServerClientPolicy(tenantCode, token string) error {
	// Get the auth-server client ID
	clientId, err := kc.getClientInternalId(tenantCode, "auth-server", token)
	if err != nil {
		return fmt.Errorf("failed to get auth-server client ID: %v", err)
	}

	// Get the realm management client ID (this manages IdP permissions)
	realmMgmtClientId, err := kc.getClientInternalId(tenantCode, "realm-management", token)
	if err != nil {
		return fmt.Errorf("failed to get realm-management client ID: %v", err)
	}

	// Create client policy for auth-server
	policyUrl := fmt.Sprintf("%s/admin/realms/%s/clients/%s/authz/resource-server/policy/client", kc.BaseURL, tenantCode, realmMgmtClientId)

	policyConfig := map[string]interface{}{
		"name":             "auth-server-token-exchange-policy",
		"description":      "Policy allowing auth-server to perform token exchange",
		"type":             "client",
		"logic":            "POSITIVE",
		"decisionStrategy": "UNANIMOUS",
		"clients":          []string{clientId},
	}

	if err := kc.makeKeycloakRequest("POST", policyUrl, policyConfig, token, "create auth-server client policy"); err != nil {
		return fmt.Errorf("failed to create client policy: %v", err)
	}

	// Get the token exchange permission ID for citizen IdP
	permissionId, err := kc.getTokenExchangePermissionId(tenantCode, realmMgmtClientId, token)
	if err != nil {
		return fmt.Errorf("failed to get token exchange permission ID: %v", err)
	}

	// Attach the policy to the permission
	attachUrl := fmt.Sprintf("%s/admin/realms/%s/clients/%s/authz/resource-server/permission/scope/%s", kc.BaseURL, tenantCode, realmMgmtClientId, permissionId)

	// Get current permission to update it
	currentPermission, err := kc.getCurrentPermission(attachUrl, token)
	if err != nil {
		return fmt.Errorf("failed to get current permission: %v", err)
	}

	// Add our policy to existing policies
	policies := []string{}
	if existingPolicies, ok := currentPermission["policies"].([]interface{}); ok {
		for _, policy := range existingPolicies {
			if policyStr, ok := policy.(string); ok {
				policies = append(policies, policyStr)
			}
		}
	}
	policies = append(policies, "auth-server-token-exchange-policy")

	// Update permission with new policy
	updateConfig := map[string]interface{}{
		"name":             currentPermission["name"],
		"description":      currentPermission["description"],
		"type":             currentPermission["type"],
		"logic":            currentPermission["logic"],
		"decisionStrategy": currentPermission["decisionStrategy"],
		"resources":        currentPermission["resources"],
		"scopes":           currentPermission["scopes"],
		"policies":         policies,
	}

	return kc.makeKeycloakRequest("PUT", attachUrl, updateConfig, token, "attach policy to token exchange permission")
}

// getClientInternalId retrieves the internal ID of a client by its clientId
func (kc *KeycloakClient) getClientInternalId(tenantCode, clientId, token string) (string, error) {
	url := fmt.Sprintf("%s/admin/realms/%s/clients?clientId=%s", kc.BaseURL, tenantCode, clientId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create get client request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get client: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get client, status: %d, body: %s", resp.StatusCode, string(body))
	}

	body, _ := ioutil.ReadAll(resp.Body)
	var clients []map[string]interface{}
	if err := json.Unmarshal(body, &clients); err != nil {
		return "", fmt.Errorf("failed to parse clients response: %v", err)
	}

	if len(clients) == 0 {
		return "", fmt.Errorf("client %s not found", clientId)
	}

	if id, ok := clients[0]["id"].(string); ok {
		return id, nil
	}

	return "", fmt.Errorf("client ID not found in response")
}

// getTokenExchangePermissionId finds the token exchange permission ID for citizen IdP
func (kc *KeycloakClient) getTokenExchangePermissionId(tenantCode, realmMgmtClientId, token string) (string, error) {
	url := fmt.Sprintf("%s/admin/realms/%s/clients/%s/authz/resource-server/permission", kc.BaseURL, tenantCode, realmMgmtClientId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create get permissions request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get permissions: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get permissions, status: %d, body: %s", resp.StatusCode, string(body))
	}

	body, _ := ioutil.ReadAll(resp.Body)
	var permissions []map[string]interface{}
	if err := json.Unmarshal(body, &permissions); err != nil {
		return "", fmt.Errorf("failed to parse permissions response: %v", err)
	}

	// Look for token exchange permission for the citizen IdP.
	// Keycloak uses the IdP's internal ID in the permission name, not the alias.
	for _, p := range permissions {
		if name, ok := p["name"].(string); ok {
			// The name is 'token-exchange.permission.idp.<some-id>'
			if strings.HasPrefix(name, "token-exchange.permission.idp.") {
				if id, ok := p["id"].(string); ok {
					return id, nil // Found it
				}
			}
		}
	}

	// If not found, log available permissions for debugging.
	var availablePermissions []string
	for _, p := range permissions {
		if name, ok := p["name"].(string); ok {
			availablePermissions = append(availablePermissions, name)
		}
	}

	return "", fmt.Errorf("token exchange permission for citizen IdP not found. Available permissions: %v", availablePermissions)
}

// getCurrentPermission retrieves current permission configuration
func (kc *KeycloakClient) getCurrentPermission(url, token string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get permission request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get permission: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get permission, status: %d, body: %s", resp.StatusCode, string(body))
	}

	body, _ := ioutil.ReadAll(resp.Body)
	var permission map[string]interface{}
	if err := json.Unmarshal(body, &permission); err != nil {
		return nil, fmt.Errorf("failed to parse permission response: %v", err)
	}

	return permission, nil
}

// makeKeycloakRequest is a helper method to make HTTP requests to Keycloak Admin API
func (kc *KeycloakClient) makeKeycloakRequest(method, url string, payload interface{}, token, operation string) error {
	var reqBody *bytes.Buffer
	if payload != nil {
		jsonBody, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal %s payload: %v", operation, err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = bytes.NewBuffer([]byte{})
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create %s request: %v", operation, err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to %s: %v", operation, err)
	}
	defer resp.Body.Close()

	// Check for success status codes
	successCodes := map[string][]int{
		"POST": {201, 204},
		// Keycloak can return 201 on PUT when creating/attaching a policy to a permission
		"PUT": {200, 201, 204},
		"GET": {200},
	}

	expectedCodes, exists := successCodes[method]
	if !exists {
		expectedCodes = []int{200, 201, 204}
	}

	success := false
	for _, code := range expectedCodes {
		if resp.StatusCode == code {
			success = true
			break
		}
	}

	if !success {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to %s, status: %d, body: %s", operation, resp.StatusCode, string(body))
	}

	return nil
}

// DeleteRealm deletes a Keycloak realm
func (kc *KeycloakClient) DeleteRealm(realmName string) error {
	// Get admin token
	token, err := kc.GetAdminToken()
	if err != nil {
		return fmt.Errorf("failed to get admin token: %v", err)
	}

	// Delete the realm
	url := fmt.Sprintf("%s/admin/realms/%s", kc.BaseURL, realmName)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete realm request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete realm: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 && resp.StatusCode != 404 { // 404 if realm doesn't exist
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete realm, status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}
