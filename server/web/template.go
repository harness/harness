package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"golang.org/x/net/html"
)

// default func map with json parser.
var funcMap = template.FuncMap{
	"json": func(v interface{}) template.JS {
		a, _ := json.Marshal(v)
		return template.JS(a)
	},
}

// helper function creates a new template from the text string.
func mustCreateTemplate(text string) *template.Template {
	templ, err := createTemplate(text)
	if err != nil {
		panic(err)
	}
	return templ
}

// helper function creates a new template from the text string.
func createTemplate(text string) (*template.Template, error) {
	templ, err := template.New("_").Funcs(funcMap).Parse(partials)
	if err != nil {
		return nil, err
	}
	return templ.Parse(
		injectPartials(text),
	)
}

// helper function that parses the html file and injects
// named partial templates.
func injectPartials(s string) string {
	w := new(bytes.Buffer)
	r := bytes.NewBufferString(s)
	t := html.NewTokenizer(r)
	for {
		tt := t.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt == html.CommentToken {
			txt := string(t.Text())
			txt = strings.TrimSpace(txt)
			seg := strings.Split(txt, ":")
			if len(seg) == 2 && seg[0] == "drone" {
				fmt.Fprintf(w, "{{ template %q . }}", seg[1])
				continue
			}
		}
		w.Write(t.Raw())
	}
	return w.String()
}

const partials = `
{{define "user"}}
{{ if .user }}
<script>
	window.DRONE_USER = {{ json .user }};
	window.DRONE_SYNC = {{ .syncing }};
</script>
{{ end }}
{{end}}

{{define "csrf"}}
{{ if .csrf -}}
<script>
	window.DRONE_CSRF = "{{ .csrf }}"
</script>
{{- end }}
{{end}}

{{define "version"}}
	<meta name="version" content="{{ .version }}">
{{end}}

{{define "docs"}}
{{ if .docs -}}
<script>
	window.DRONE_DOCS = "{{ .docs }}"
</script>
{{- end }}
{{end}}
`
