package template

//go:generate sh -c "amberc amber/400.amber            > amber_gen/400.html"
//go:generate sh -c "amberc amber/401.amber            > amber_gen/401.html"
//go:generate sh -c "amberc amber/403.amber            > amber_gen/403.html"
//go:generate sh -c "amberc amber/404.amber            > amber_gen/404.html"
//go:generate sh -c "amberc amber/500.amber            > amber_gen/500.html"
//go:generate sh -c "amberc amber/build.amber          > amber_gen/build.html"
//go:generate sh -c "amberc amber/login.amber          > amber_gen/login.html"
//go:generate sh -c "amberc amber/login_form.amber     > amber_gen/login_form.html"
//go:generate sh -c "amberc amber/repos.amber          > amber_gen/repos.html"
//go:generate sh -c "amberc amber/repo.amber           > amber_gen/repo.html"
//go:generate sh -c "amberc amber/repo_badge.amber     > amber_gen/repo_badge.html"
//go:generate sh -c "amberc amber/repo_activate.amber  > amber_gen/repo_activate.html"
//go:generate sh -c "amberc amber/repo_config.amber    > amber_gen/repo_config.html"
//go:generate sh -c "amberc amber/repo_secret.amber    > amber_gen/repo_secret.html"
//go:generate sh -c "amberc amber/users.amber          > amber_gen/users.html"
//go:generate sh -c "amberc amber/user.amber           > amber_gen/user.html"
//go:generate sh -c "amberc amber/nodes.amber          > amber_gen/nodes.html"
//go:generate sh -c "amberc amber/index.amber          > amber_gen/index.html"

//go:generate go-bindata -pkg template -o template_gen.go amber_gen/

import (
	"encoding/json"
	"html/template"
	"path/filepath"

	"github.com/eknkc/amber"
)

func Load() *template.Template {
	amber.FuncMap["json"] = marshal

	dir, _ := AssetDir("amber_gen")
	tmpl := template.New("_")
	tmpl.Funcs(amber.FuncMap)

	for _, name := range dir {
		if filepath.Ext(name) != ".html" {
			continue
		}

		path := filepath.Join("amber_gen", name)
		src := MustAsset(path)
		tmpl = template.Must(
			tmpl.New(name).Parse(string(src)),
		)
	}

	return tmpl
}

// marshal is a helper function to render data as JSON
// inside the template.
func marshal(v interface{}) template.JS {
	a, _ := json.Marshal(v)
	return template.JS(a)
}
