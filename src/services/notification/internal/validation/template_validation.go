package validation

import (
	"fmt"
	htmlTemplate "html/template"
	"notification/internal/models"
	"strings"
	textTemplate "text/template"
)

type TemplateValidator struct{}

func NewTemplateValidator() *TemplateValidator {
	return &TemplateValidator{}
}

func (v *TemplateValidator) ValidateTemplate(tmpl *models.Template) error {
	var errors []string

	content := strings.TrimSpace(tmpl.Content)
	subject := strings.TrimSpace(tmpl.Subject)

	switch tmpl.Type {
	case models.TemplateTypeEmail:
		if subject == "" {
			errors = append(errors, "subject is required for EMAIL templates")
		} else if err := validateGoTemplate(subject, false); err != nil {
			errors = append(errors, "invalid subject Go template syntax: "+err.Error())
		}

		if content == "" {
			errors = append(errors, "content is required for EMAIL templates")
		} else if err := validateGoTemplate(content, tmpl.IsHTML); err != nil {
			errors = append(errors, "invalid content Go template syntax: "+err.Error())
		}

	case models.TemplateTypeSMS:
		if content == "" {
			errors = append(errors, "content is required for SMS templates")
		} else if tmpl.IsHTML {
			errors = append(errors, "content for SMS templates cannot be HTML")
		} else if err := validateGoTemplate(content, false); err != nil {
			errors = append(errors, "invalid content Go template syntax: "+err.Error())
		}

	default:
		errors = append(errors, fmt.Sprintf("unsupported template type: %s", tmpl.Type))
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errors, "; "))
	}
	return nil
}

func validateGoTemplate(tmplStr string, isHTML bool) error {
	if isHTML {
		_, err := htmlTemplate.New("html_check").Parse(tmplStr)
		return err
	}
	_, err := textTemplate.New("text_check").Parse(tmplStr)
	return err
}
