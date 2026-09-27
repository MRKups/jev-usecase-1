package webapp

import (
	"bytes"
	"embed"
	"html/template"
)

//go:embed static
var staticFS embed.FS

//go:embed templates/*.html
var templateFS embed.FS

var (
	dashboardTemplate = template.Must(template.ParseFS(templateFS, "templates/*.html"))
	IndexHTML         = renderDashboardHTML()
)

func renderDashboardHTML() string {
	var buf bytes.Buffer
	if err := dashboardTemplate.ExecuteTemplate(&buf, "index.html", nil); err != nil {
		panic("failed to render dashboard template: " + err.Error())
	}
	return buf.String()
}
