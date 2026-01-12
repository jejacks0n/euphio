package ansi

import (
	"bytes"
	"text/template"

	"euphio/internal/app"

	"github.com/Masterminds/sprig/v3"
)

// TemplateData holds the data available to ANSI/art templates.
type TemplateData struct {
	BoardName       string
	PrettyBoardName string
	Description     string
	Hostname        string
	Website         string
	Version         string
	// We can add more fields here as needed, e.g., User info
	Custom map[string]interface{}
}

// NewTemplateData creates a TemplateData struct populated with global config values.
func NewTemplateData() *TemplateData {
	return &TemplateData{
		BoardName:       app.Config.General.BoardName,
		PrettyBoardName: app.Config.General.PrettyBoardName,
		Description:     app.Config.General.Description,
		Hostname:        app.Config.General.Hostname,
		Website:         app.Config.General.Website,
		Version:         app.Version,
		Custom:          make(map[string]interface{}),
	}
}

// RenderTemplate parses and executes the given data as a Go template.
// It automatically injects global configuration values.
// You can provide additional custom data via the 'extra' map.
func RenderTemplate(data []byte, extra map[string]interface{}) ([]byte, error) {
	tmplData := NewTemplateData()

	// Merge extra data
	for k, v := range extra {
		tmplData.Custom[k] = v
	}

	// Create template with Sprig functions
	tmpl, err := template.New("ansi").Funcs(sprig.FuncMap()).Parse(string(data))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, tmplData); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
