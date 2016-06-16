package template

//go:generate go-bindata -pkg template -o template_gen.go files/

import (
	"encoding/json"
	"html/template"
	"path/filepath"
)

// Load loads the templates from the embedded file map. This function will not
// compile if go generate is not executed before.
func Load() *template.Template {
	dir, _ := AssetDir("files")
	tmpl := template.New("_").Funcs(template.FuncMap{"json": marshal})
	for _, name := range dir {
		path := filepath.Join("files", name)
		src := MustAsset(path)
		tmpl = template.Must(
			tmpl.New(name).Parse(string(src)),
		)
	}

	return tmpl
}

// marshal is a helper function to render data as JSON inside the template.
func marshal(v interface{}) template.JS {
	a, _ := json.Marshal(v)
	return template.JS(a)
}
