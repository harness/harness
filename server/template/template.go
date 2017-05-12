package template

//go:generate togo tmpl -package template -func funcMap -format html -input files/*.html

import (
	"encoding/json"
	"html/template"
)

var funcMap = template.FuncMap{"json": marshal}

// marshal is a helper function to render data as JSON inside the template.
func marshal(v interface{}) template.JS {
	a, _ := json.Marshal(v)
	return template.JS(a)
}
