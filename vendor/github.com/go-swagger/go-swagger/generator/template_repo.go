package generator

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"text/template/parse"

	"bitbucket.org/pkg/inflect"
	"github.com/go-swagger/go-swagger/swag"
)

var protectedTemplates = map[string]bool{
	"schemabody":                     true,
	"privtuplefield":                 true,
	"withoutBaseTypeBody":            true,
	"swaggerJsonEmbed":               true,
	"validationCustomformat":         true,
	"tuplefield":                     true,
	"header":                         true,
	"withBaseTypeBody":               true,
	"primitivefieldvalidator":        true,
	"mapvalidator":                   true,
	"propertyValidationDocString":    true,
	"typeSchemaType":                 true,
	"docstring":                      true,
	"dereffedSchemaType":             true,
	"model":                          true,
	"modelvalidator":                 true,
	"privstructfield":                true,
	"schemavalidator":                true,
	"tuplefieldIface":                true,
	"tupleSerializer":                true,
	"tupleserializer":                true,
	"propertyvalidator":              true,
	"structfieldIface":               true,
	"schemaBody":                     true,
	"objectvalidator":                true,
	"schematype":                     true,
	"additionalpropertiesserializer": true,
	"slicevalidator":                 true,
	"validationStructfield":          true,
	"validationPrimitive":            true,
	"schemaType":                     true,
	"subTypeBody":                    true,
	"schema":                         true,
	"additionalPropertiesSerializer": true,
	"serverDoc":                      true,
	"structfield":                    true,
	"hasDiscriminatedSerializer":     true,
	"discriminatedSerializer":        true,
}

// FuncMap is a map with default functions for use n the templates.
// These are available in every template
var FuncMap template.FuncMap = map[string]interface{}{
	"pascalize": func(arg string) string {
		if len(arg) == 0 || arg[0] > '9' {
			return swag.ToGoName(arg)
		}

		return swag.ToGoName("Nr " + arg)
	},
	"camelize":  swag.ToJSONName,
	"humanize":  swag.ToHumanNameLower,
	"snakize":   swag.ToFileName,
	"dasherize": swag.ToCommandName,
	"pluralizeFirstWord": func(arg string) string {
		sentence := strings.Split(arg, " ")
		if len(sentence) == 1 {
			return inflect.Pluralize(arg)
		}

		return inflect.Pluralize(sentence[0]) + " " + strings.Join(sentence[1:], " ")
	},
	"json":       asJSON,
	"prettyjson": asPrettyJSON,
	"hasInsecure": func(arg []string) bool {
		return swag.ContainsStringsCI(arg, "http") || swag.ContainsStringsCI(arg, "ws")
	},
	"hasSecure": func(arg []string) bool {
		return swag.ContainsStringsCI(arg, "https") || swag.ContainsStringsCI(arg, "wss")
	},
	"stripPackage": func(str, pkg string) string {
		parts := strings.Split(str, ".")
		strlen := len(parts)
		if strlen > 0 {
			return parts[strlen-1]
		}
		return str
	},
	"dropPackage": func(str string) string {
		parts := strings.Split(str, ".")
		strlen := len(parts)
		if strlen > 0 {
			return parts[strlen-1]
		}
		return str
	},
	"upper": func(str string) string {
		return strings.ToUpper(str)
	},
	"contains": func(coll []string, arg string) bool {
		for _, v := range coll {
			if v == arg {
				return true
			}
		}
		return false
	},
}

// NewRepository creates a new template repository with the provided functions defined
func NewRepository(funcs template.FuncMap) *Repository {
	repo := Repository{
		files:     make(map[string]string),
		templates: make(map[string]*template.Template),
		funcs:     funcs,
	}

	if repo.funcs == nil {

		repo.funcs = make(template.FuncMap)
	}

	return &repo
}

// Repository is the repository for the generator templates.
type Repository struct {
	files     map[string]string
	templates map[string]*template.Template
	funcs     template.FuncMap
}

// LoadDefaults will load the embedded templates
func (t *Repository) LoadDefaults() {

	for name, asset := range assets {
		t.addFile(name, string(asset), true)
	}
}

// LoadDir will walk the specified path and add each .gotmpl file it finds to the repository
func (t *Repository) LoadDir(templatePath string) error {

	err := filepath.Walk(templatePath, func(path string, info os.FileInfo, err error) error {

		if strings.HasSuffix(path, ".gotmpl") {
			assetName := strings.TrimPrefix(path, templatePath)
			if data, err := ioutil.ReadFile(path); err == nil {
				t.AddFile(assetName, string(data))
			}
		}
		if err != nil {
			return err
		}

		return nil
	})

	return err
}

func (t *Repository) addFile(name, data string, allowOverride bool) error {
	fileName := name
	name = swag.ToJSONName(strings.TrimSuffix(name, ".gotmpl"))

	templ, err := template.New(name).Funcs(t.funcs).Parse(data)

	if err != nil {
		return err
	}

	// check if any protected templates are defined
	if !allowOverride {
		for _, template := range templ.Templates() {
			if protectedTemplates[template.Name()] {
				return fmt.Errorf("Cannot overwrite protected template %s", template.Name())
			}
		}
	}

	// Add each defined tempalte into the cache
	for _, template := range templ.Templates() {

		t.files[template.Name()] = fileName
		t.templates[template.Name()] = template.Lookup(template.Name())
	}

	return nil
}

// AddFile adds a file to the repository. It will create a new template based on the filename.
// It trims the .gotmpl from the end and converts the name using swag.ToJSONName. This will strip
// directory separators and Camelcase the next letter.
// e.g validation/primitive.gotmpl will become validationPrimitive
//
// If the file contains a definition for a template that is protected the whole file will not be added
func (t *Repository) AddFile(name, data string) error {
	return t.addFile(name, data, false)
}

func findDependencies(n parse.Node) []string {

	var deps []string
	depMap := make(map[string]bool)

	if n == nil {
		return deps
	}

	switch node := n.(type) {
	case *parse.ListNode:
		if node != nil && node.Nodes != nil {
			for _, nn := range node.Nodes {
				for _, dep := range findDependencies(nn) {
					depMap[dep] = true
				}
			}
		}
	case *parse.IfNode:
		for _, dep := range findDependencies(node.BranchNode.List) {
			depMap[dep] = true
		}
		for _, dep := range findDependencies(node.BranchNode.ElseList) {
			depMap[dep] = true
		}

	case *parse.RangeNode:
		for _, dep := range findDependencies(node.BranchNode.List) {
			depMap[dep] = true
		}
		for _, dep := range findDependencies(node.BranchNode.ElseList) {
			depMap[dep] = true
		}

	case *parse.WithNode:
		for _, dep := range findDependencies(node.BranchNode.List) {
			depMap[dep] = true
		}
		for _, dep := range findDependencies(node.BranchNode.ElseList) {
			depMap[dep] = true
		}

	case *parse.TemplateNode:
		depMap[node.Name] = true
	}

	for dep := range depMap {
		deps = append(deps, dep)
	}

	return deps

}

func (t *Repository) flattenDependencies(templ *template.Template, dependencies map[string]bool) map[string]bool {
	if dependencies == nil {
		dependencies = make(map[string]bool)
	}

	deps := findDependencies(templ.Tree.Root)

	for _, d := range deps {
		if _, found := dependencies[d]; !found {

			dependencies[d] = true

			if tt := t.templates[d]; tt != nil {
				dependencies = t.flattenDependencies(tt, dependencies)
			}
		}

		dependencies[d] = true

	}

	return dependencies

}

func (t *Repository) addDependencies(templ *template.Template) (*template.Template, error) {

	name := templ.Name()

	deps := t.flattenDependencies(templ, nil)

	for dep := range deps {

		if dep == "" {
			continue
		}

		tt := templ.Lookup(dep)

		// Check if we have it
		if tt == nil {
			tt = t.templates[dep]

			// Still dont have it return an error
			if tt == nil {
				return templ, fmt.Errorf("Could not find template %s", dep)
			}
			var err error

			// Add it to the parse tree
			templ, err = templ.AddParseTree(dep, tt.Tree)

			if err != nil {
				return templ, fmt.Errorf("Dependency Error: %v", err)
			}

		}
	}
	return templ.Lookup(name), nil
}

// Get will return the named template from the repository, ensuring that all dependent templates are loaded.
// It will return an error if a dependent template is not defined in the repository.
func (t *Repository) Get(name string) (*template.Template, error) {
	templ, found := t.templates[name]

	if !found {
		return templ, fmt.Errorf("Template doesn't exist %s", name)
	}

	return t.addDependencies(templ)
}

// DumpTemplates prints out a dump of all the defined templates, where they are defined and what their dependencies are.
func (t *Repository) DumpTemplates() {

	fmt.Println("# Templates")
	for name, templ := range t.templates {
		fmt.Printf("## %s\n", name)
		fmt.Printf("Defined in `%s`\n", t.files[name])

		if deps := findDependencies(templ.Tree.Root); len(deps) > 0 {

			fmt.Printf("####requires \n - %v\n\n\n", strings.Join(deps, "\n - "))
		}
		fmt.Println("\n---")
	}
}
