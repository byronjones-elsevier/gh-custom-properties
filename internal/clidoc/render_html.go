package clidoc

import (
	"html/template"
	"strings"
)

var htmlTemplate = template.Must(template.New("help").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>{{.Name}} {{.Version}}</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; max-width: 48rem; margin: 2rem auto; padding: 0 1rem; color: #222; }
  h1 { font-size: 1.4rem; }
  h2 { font-size: 1.05rem; margin-top: 2rem; border-bottom: 1px solid #ddd; padding-bottom: .25rem; }
  code, pre { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; background: #f5f5f5; }
  pre { padding: .75rem 1rem; overflow-x: auto; }
  table { border-collapse: collapse; width: 100%; }
  td { vertical-align: top; padding: .3rem .6rem .3rem 0; }
  td.flag { white-space: nowrap; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
</style>
</head>
<body>
<h1>{{.Name}} <small>{{.Version}}</small></h1>
<p>{{.Summary}}</p>

<h2>Usage</h2>
<pre>{{range .Usage}}{{.}}
{{end}}</pre>
{{if .Note}}<p>{{.Note}}</p>{{end}}

<h2>Flags</h2>
<table>
{{range .Flags}}<tr><td class="flag">{{.FlagLabel}}</td><td>{{.Description}}</td></tr>
{{end}}</table>

{{range .Section}}<h2>{{.Title}}</h2>
<pre>{{.Body}}</pre>
{{end}}
</body>
</html>
`))

// RenderHTML renders the spec as a standalone HTML help page.
func (s Spec) RenderHTML() (string, error) {
	var b strings.Builder
	if err := htmlTemplate.Execute(&b, s); err != nil {
		return "", err
	}
	return b.String(), nil
}
