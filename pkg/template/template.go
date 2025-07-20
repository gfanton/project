package template

import (
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed *.init
var templates embed.FS

// Data holds template data for shell initialization.
type Data struct {
	Exec string // Path to the project executable
}

// Render renders the specified template with the given data.
func Render(name string, data Data) (string, error) {
	tmplData, err := templates.ReadFile(name + ".init")
	if err != nil {
		return "", fmt.Errorf("failed to read template %s: %w", name, err)
	}

	tmpl, err := template.New(name).Parse(string(tmplData))
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return buf.String(), nil
}
