package config

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"
)

type Config struct {
	Port string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	Vault         VaultConfig
	VaultRequired bool
	IDGen         IDGenConfig
	ExternalRegs  map[string]ExternalRegistryConfig
}

type VaultConfig struct {
	Address      string
	Token        string
	TransitMount string
	KeyPrefix    string
	Timeout      time.Duration
}

type IDGenConfig struct {
	BaseURL      string
	TemplateCode string
	OrgValue     string
	HTTPTimeout  time.Duration
}

type ExternalRegistryConfig struct {
	BaseURL    string            `json:"baseUrl"`
	Headers    map[string]string `json:"headers"`
	Timeout    time.Duration     `json:"-"`
	TimeoutRaw string            `json:"timeout"`
}

func Load() *Config {
	cfg := &Config{
		Port: getEnv("PORT", "8080"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "registry_user"),
		DBPassword: getEnv("DB_PASSWORD", "registry_password"),
		DBName:     getEnv("DB_NAME", "registry_db"),
		DBSSLMode:  getEnv("DB_SSL_MODE", "disable"),

		Vault: VaultConfig{
			Address:      getEnv("VAULT_ADDRESS", ""),
			Token:        getEnv("VAULT_TOKEN", ""),
			TransitMount: getEnv("VAULT_TRANSIT_MOUNT", "transit"),
			KeyPrefix:    getEnv("VAULT_KEY_PREFIX", "registry-"),
			Timeout:      parseDuration(getEnv("VAULT_TIMEOUT", "")),
		},
		VaultRequired: getEnvAsBool("VAULT_REQUIRED", false),

		IDGen: IDGenConfig{
			BaseURL:      getEnv("IDGEN_BASE_URL", "http://localhost:8090/idgen"),
			TemplateCode: getEnv("IDGEN_TEMPLATE_ID", "registryId"),
			OrgValue:     getEnv("IDGEN_ORG_VALUE", "REGISTRY"),
			HTTPTimeout:  parseDuration(getEnv("IDGEN_HTTP_TIMEOUT", "5s")),
		},

		ExternalRegs: loadRegistryDiscovery(),
	}

	return cfg
}

func loadRegistryDiscovery() map[string]ExternalRegistryConfig {
	if raw := getEnv("REGISTRY_DISCOVERY", ""); strings.TrimSpace(raw) != "" {
		return parseRegistryDiscovery(raw)
	}
	filePath := strings.TrimSpace(getEnv("REGISTRY_DISCOVERY_FILE", ""))
	if filePath == "" {
		return loadRegistryDiscoveryFromEnvKeys()
	}
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return loadRegistryDiscoveryFromEnvKeys()
	}
	if parsed := parseRegistryDiscovery(string(bytes)); parsed != nil {
		return parsed
	}
	return loadRegistryDiscoveryFromEnvKeys()
}

func loadRegistryDiscoveryFromEnvKeys() map[string]ExternalRegistryConfig {
	keysRaw := strings.TrimSpace(getEnv("REGISTRY_DISCOVERY_KEYS", ""))
	if keysRaw == "" {
		return nil
	}
	keys := strings.Split(keysRaw, ",")
	result := make(map[string]ExternalRegistryConfig)
	for _, key := range keys {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		envPrefix := strings.ToUpper(strings.ReplaceAll(trimmed, "-", "_"))
		baseURL := strings.TrimSpace(getEnv(fmt.Sprintf("REGISTRY_DISCOVERY_%s_BASE_URL", envPrefix), ""))
		if baseURL == "" {
			continue
		}
		headersRaw := getEnv(fmt.Sprintf("REGISTRY_DISCOVERY_%s_HEADERS", envPrefix), "")
		headers := map[string]string{}
		if strings.TrimSpace(headersRaw) != "" {
			_ = json.Unmarshal([]byte(headersRaw), &headers)
		}
		timeout := parseDuration(getEnv(fmt.Sprintf("REGISTRY_DISCOVERY_%s_TIMEOUT", envPrefix), ""))
		if timeout == 0 {
			timeout = 5 * time.Second
		}
		result[trimmed] = ExternalRegistryConfig{
			BaseURL: baseURL,
			Headers: headers,
			Timeout: timeout,
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func parseRegistryDiscovery(raw string) map[string]ExternalRegistryConfig {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var parsed map[string]ExternalRegistryConfig
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil
	}
	for name, cfg := range parsed {
		if cfg.Headers == nil {
			cfg.Headers = map[string]string{}
		}
		if cfg.Timeout == 0 {
			if strings.TrimSpace(cfg.TimeoutRaw) != "" {
				if d, err := time.ParseDuration(cfg.TimeoutRaw); err == nil {
					cfg.Timeout = d
				}
			}
		}
		if cfg.Timeout == 0 {
			cfg.Timeout = 5 * time.Second
		}
		cfg.TimeoutRaw = ""
		parsed[name] = cfg
	}
	return parsed
}

func (c *Config) DatabaseURL() string {
	u := &url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(c.DBHost, c.DBPort),
		Path:   "/" + c.DBName,
	}

	if c.DBUser != "" {
		if c.DBPassword != "" {
			u.User = url.UserPassword(c.DBUser, c.DBPassword)
		} else {
			u.User = url.User(c.DBUser)
		}
	}

	query := u.Query()
	if c.DBSSLMode != "" {
		query.Set("sslmode", c.DBSSLMode)
	}
	u.RawQuery = query.Encode()

	return u.String()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(value string) time.Duration {
	if value == "" {
		return 0
	}
	if d, err := time.ParseDuration(value); err == nil {
		return d
	}
	return 0
}

func getEnvAsBool(key string, defaultValue bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	switch strings.ToLower(value) {
	case "true", "1", "yes", "y":
		return true
	case "false", "0", "no", "n":
		return false
	default:
		return defaultValue
	}
}
