package email

import (
	"bytes"
	"embed"
	"fmt"
	htmltemplate "html/template"
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
		"formatTime": formatTemplateTime,
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
		"formatTime": formatTemplateTime,
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
