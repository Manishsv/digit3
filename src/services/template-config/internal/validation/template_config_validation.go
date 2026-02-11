package validation

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"template-config/internal/models"
)

type TemplateValidator struct{}

func NewTemplateValidator() *TemplateValidator {
	return &TemplateValidator{}
}

// ValidateTemplateConfig validates the full TemplateConfig object.
func (v *TemplateValidator) ValidateTemplateConfig(config *models.TemplateConfig) error {
	// Validate field mappings
	if err := validateStringMap("fieldMapping", config.FieldMapping, false); err != nil {
		return err
	}

	// Validate API mappings
	return v.validateAPIMappings(config.APIMapping)
}

//
// ---- API Mappings ----
//

func (v *TemplateValidator) validateAPIMappings(mappings []models.APIMapping) error {
	for i, mapping := range mappings {
		prefix := fmt.Sprintf("apiMapping[%d]", i)

		// 1. Validate HTTP method
		if mapping.Method != "GET" {
			return fmt.Errorf("%s: only GET method is allowed", prefix)
		}

		// 2. Validate base URL
		if err := validateBase(mapping.Endpoint.Base); err != nil {
			return fmt.Errorf("%s: invalid base URL: %w", prefix, err)
		}

		// 3. Validate path
		if err := validatePath(mapping.Endpoint.Base, mapping.Endpoint.Path); err != nil {
			return fmt.Errorf("%s: invalid path: %w", prefix, err)
		}

		// 4. Validate path params exist for placeholders
		if err := validatePathParams(mapping.Endpoint.Path, mapping.Endpoint.PathParams); err != nil {
			return fmt.Errorf("%s: %w", prefix, err)
		}

		// 5. Validate query params
		if err := validateStringMap("queryParams", mapping.Endpoint.QueryParams, false); err != nil {
			return fmt.Errorf("%s: %w", prefix, err)
		}

		// 6. Validate response mapping
		if err := validateStringMap("responseMapping", mapping.ResponseMapping, false); err != nil {
			return fmt.Errorf("%s: %w", prefix, err)
		}
	}
	return nil
}

//
// ---- Reusable Map Validation ----
//

func validateStringMap(name string, m map[string]string, allowPlaceholders bool) error {
	for key, value := range m {
		if key == "" {
			return fmt.Errorf("%s key cannot be empty", name)
		}
		if !allowPlaceholders && strings.Contains(key, "{{") {
			return fmt.Errorf("placeholders not allowed in %s key: %s", name, key)
		}
		if value == "" {
			return fmt.Errorf("%s key '%s' has empty value", name, key)
		}
		if !isValidJSONPath(value) {
			return fmt.Errorf("invalid JSONPath for key '%s': %s", key, value)
		}
	}
	return nil
}

func isValidJSONPath(s string) bool {
	return strings.HasPrefix(s, "$")
}

//
// ---- Base and Path Validation ----
//

func validateBase(base string) error {
	if strings.Contains(base, "{{") || strings.Contains(base, "}}") {
		return errors.New("placeholders not allowed in base URL")
	}

	if strings.HasSuffix(base, "/") {
		return errors.New("base URL must not end with a '/'")
	}

	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return errors.New("must be a valid absolute URL with scheme and host")
	}
	return nil
}

func validatePath(base string, path string) error {
	if !strings.HasPrefix(path, "/") {
		return errors.New("path must start with '/'")
	}

	re := regexp.MustCompile(`\{\{[^}]+\}\}`)
	safePath := re.ReplaceAllString(path, "placeholder")

	u, err := url.Parse(base + safePath)
	if err != nil || u.Path == "" {
		return fmt.Errorf("invalid path format: %s", base+path)
	}
	return nil
}

func validatePathParams(path string, pathParams map[string]string) error {
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	matches := re.FindAllStringSubmatch(path, -1)

	for _, match := range matches {
		param := match[1]
		if _, ok := pathParams[param]; !ok {
			return fmt.Errorf("missing path param: %s", param)
		}
	}

	// Validate the pathParams map values
	return validateStringMap("pathParams", pathParams, false)
}
