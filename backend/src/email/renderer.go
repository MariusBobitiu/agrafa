package email

import (
	"bytes"
	"embed"
	"fmt"
	htmltemplate "html/template"
	"strings"
	texttemplate "text/template"
	"time"
)

//go:embed templates/*.html templates/*.txt
var templateFS embed.FS

type Renderer struct{}

func NewRenderer() *Renderer {
	return &Renderer{}
}

func (r *Renderer) RenderHTML(templateName string, data any) (string, error) {
	tmpl, err := htmltemplate.New(templateName).Funcs(htmltemplate.FuncMap{
		"formatTime":    formatTemplateTime,
		"formatNumber":  formatTemplateNumber,
		"formatPercent": formatTemplatePercent,
		"titleCase":     titleCase,
		"upper":         strings.ToUpper,
		"plainTitle":    plainAlertTitle,
	}).ParseFS(templateFS, "templates/"+templateName)
	if err != nil {
		return "", fmt.Errorf("parse html email template %q: %w", templateName, err)
	}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, data); err != nil {
		return "", fmt.Errorf("render html email template %q: %w", templateName, err)
	}

	return output.String(), nil
}

func (r *Renderer) RenderText(templateName string, data any) (string, error) {
	tmpl, err := texttemplate.New(templateName).Funcs(texttemplate.FuncMap{
		"formatTime":    formatTemplateTime,
		"formatNumber":  formatTemplateNumber,
		"formatPercent": formatTemplatePercent,
		"titleCase":     titleCase,
		"upper":         strings.ToUpper,
		"plainTitle":    plainAlertTitle,
	}).ParseFS(templateFS, "templates/"+templateName)
	if err != nil {
		return "", fmt.Errorf("parse text email template %q: %w", templateName, err)
	}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, data); err != nil {
		return "", fmt.Errorf("render text email template %q: %w", templateName, err)
	}

	return output.String(), nil
}

func plainAlertTitle(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "⚠ ")
	return strings.TrimPrefix(value, "✓ ")
}

func formatTemplateNumber(value any) string {
	switch typed := value.(type) {
	case float64:
		return fmt.Sprintf("%g", typed)
	case *float64:
		if typed != nil {
			return fmt.Sprintf("%g", *typed)
		}
	}
	return ""
}

func formatTemplatePercent(value any) string {
	formatted := formatTemplateNumber(value)
	if formatted == "" {
		return ""
	}
	return formatted + "%"
}

func formatTemplateTime(value any) string {
	switch typed := value.(type) {
	case time.Time:
		if typed.IsZero() {
			return ""
		}

		return typed.UTC().Format(time.RFC1123)
	case *time.Time:
		if typed == nil || typed.IsZero() {
			return ""
		}

		return typed.UTC().Format(time.RFC1123)
	default:
		return ""
	}
}
