package service

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	texttemplate "text/template"
)

type TemplateRenderer struct{}

func NewTemplateRenderer() *TemplateRenderer {
	return &TemplateRenderer{}
}

// RenderTemplate renders a template using Go templates with the provided data
func (r *TemplateRenderer) RenderTemplate(templateContent string, data map[string]interface{}, isHTML bool) (string, error) {
	var tmpl interface{}
	var err error

	if isHTML {
		tmpl, err = htmltemplate.New("template").Parse(templateContent)
	} else {
		tmpl, err = texttemplate.New("template").Parse(templateContent)
	}

	if err != nil {
		return "", fmt.Errorf("failed to parse template: %v", err)
	}

	var buf bytes.Buffer
	if htmlTmpl, ok := tmpl.(*htmltemplate.Template); ok {
		if err := htmlTmpl.Execute(&buf, data); err != nil {
			return "", fmt.Errorf("failed to execute HTML template: %v", err)
		}
	} else if textTmpl, ok := tmpl.(*texttemplate.Template); ok {
		if err := textTmpl.Execute(&buf, data); err != nil {
			return "", fmt.Errorf("failed to execute text template: %v", err)
		}
	}

	return buf.String(), nil
}

// RenderSubject renders the subject template if it exists
func (r *TemplateRenderer) RenderSubject(subjectTemplate string, data map[string]interface{}) (string, error) {
	if subjectTemplate == "" {
		return "", nil
	}

	return r.RenderTemplate(subjectTemplate, data, false)
}

// RenderContent renders the content template
func (r *TemplateRenderer) RenderContent(contentTemplate string, data map[string]interface{}, isHTML bool) (string, error) {
	return r.RenderTemplate(contentTemplate, data, isHTML)
}
