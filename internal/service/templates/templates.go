package templates

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"text/template"
)

//go:embed *.gotmpl
var templatesFS embed.FS

// Renderer is a template renderer.
type Renderer interface {
	Help(params HelpParams) (string, error)
	Welcome(params WelcomeParams) (string, error)
}

// New creates a new Renderer.
func New() (Renderer, error) {
	var errs error

	helpTpl, err := parseTemplate("help", "help.gotmpl")
	if err != nil {
		errs = errors.Join(errs, err)
	}

	welcomeTpl, err := parseTemplate("welcome", "welcome.gotmpl")
	if err != nil {
		errs = errors.Join(errs, err)
	}

	if errs != nil {
		return nil, errs
	}

	t := templates{
		help:    helpTpl,
		welcome: welcomeTpl,
	}

	return &t, nil
}

// templates is a template renderer.
type templates struct {
	help    *template.Template
	welcome *template.Template
}

// HelpParams is a set of parameters for Help template.
type HelpParams struct {
	BotUsername string
	Commands    string
	HelpCmd     string
}

// Help renders a help message.
func (t *templates) Help(params HelpParams) (string, error) {
	return renderTemplate(t.help, params)
}

// WelcomeParams is a set of parameters for Welcome template.
type WelcomeParams struct {
	Firstname   string
	BotUsername string
	HelpCmd     string
}

// Welcome renders a welcome message.
func (t *templates) Welcome(params WelcomeParams) (string, error) {
	return renderTemplate(t.welcome, params)
}

// renderTemplate renders a template.
func renderTemplate(tmpl *template.Template, data interface{}) (string, error) {
	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// parseTemplate parses a template from the templatesFS.
func parseTemplate(name, path string) (*template.Template, error) {
	tmplBytes, err := templatesFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %q template file: %w", name, err)
	}

	tmpl, err := template.New(name).Parse(string(tmplBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to parse %q template: %w", name, err)
	}
	return tmpl, nil
}
