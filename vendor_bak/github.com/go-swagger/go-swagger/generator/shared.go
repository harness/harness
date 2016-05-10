// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package generator

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-swagger/go-swagger/spec"
	"github.com/go-swagger/go-swagger/swag"
	"golang.org/x/tools/imports"
)

// Debug when the env var DEBUG is not empty
// the generators will be very noisy about what they are doing
var Debug = os.Getenv("DEBUG") != ""

var reservedGoWords = []string{
	"break", "default", "func", "interface", "select",
	"case", "defer", "go", "map", "struct",
	"chan", "else", "goto", "package", "switch",
	"const", "fallthrough", "if", "range", "type",
	"continue", "for", "import", "return", "var",
}

var defaultGoImports = []string{
	"bool", "int", "int8", "int16", "int32", "int64",
	"uint", "uint8", "uint16", "uint32", "uint64",
	"float32", "float64", "interface{}", "string",
	"byte", "rune",
}

var reservedGoWordSet map[string]struct{}

func init() {
	reservedGoWordSet = make(map[string]struct{})
	for _, gw := range reservedGoWords {
		reservedGoWordSet[gw] = struct{}{}
	}
}

func mangleName(name, suffix string) string {
	if _, ok := reservedGoWordSet[swag.ToFileName(name)]; !ok {
		return name
	}
	return strings.Join([]string{name, suffix}, "_")
}

func findSwaggerSpec(name string) (string, error) {
	f, err := os.Stat(name)
	if err != nil {
		return "", err
	}
	if f.IsDir() {
		return "", fmt.Errorf("%s is a directory", name)
	}
	return name, nil
}

// GenOpts the options for the generator
type GenOpts struct {
	Spec              string
	APIPackage        string
	ModelPackage      string
	ServerPackage     string
	ClientPackage     string
	Principal         string
	Target            string
	TypeMapping       map[string]string
	Imports           map[string]string
	DumpData          bool
	DefaultScheme     string
	DefaultProduces   string
	DefaultConsumes   string
	IncludeModel      bool
	IncludeValidator  bool
	IncludeHandler    bool
	IncludeParameters bool
	IncludeResponses  bool
	IncludeMain       bool
	IncludeSupport    bool
	ExcludeSpec       bool
	TemplateDir       string
	WithContext       bool
}

type generatorOptions struct {
	ModelPackage    string
	TargetDirectory string
}

func loadSpec(specFile string) (string, *spec.Document, error) {
	// find swagger spec document, verify it exists
	specPath := specFile
	var err error
	if !strings.HasPrefix(specPath, "http") {
		specPath, err = findSwaggerSpec(specFile)
		if err != nil {
			return "", nil, err
		}
	}

	// load swagger spec
	specDoc, err := spec.Load(specPath)
	if err != nil {
		return "", nil, err
	}
	return specPath, specDoc, nil
}

func fileExists(target, name string) bool {
	ffn := swag.ToFileName(name) + ".go"
	_, err := os.Stat(filepath.Join(target, ffn))
	return !os.IsNotExist(err)
}

func writeToFileIfNotExist(target, name string, content []byte) error {
	if fileExists(target, name) {
		return nil
	}
	return writeToFile(target, name, content)
}

func formatGoFile(ffn string, content []byte) ([]byte, error) {
	opts := new(imports.Options)
	opts.TabIndent = true
	opts.TabWidth = 2
	opts.Fragment = true
	opts.Comments = true

	return imports.Process(ffn, content, opts)
}

func stripTestFromFileName(name string) string {
	ffn := swag.ToFileName(name)
	if strings.HasSuffix(ffn, "_test") {
		ffn = ffn[:len(ffn)-5]
	}
	return ffn
}

func writeToFile(target, name string, content []byte) error {
	ffn := stripTestFromFileName(name) + ".go"

	res, err := formatGoFile(filepath.Join(target, ffn), content)
	if err != nil {
		log.Println(err)
		return writeFile(target, ffn, content)
	}

	return writeFile(target, ffn, res)
}

func writeToTestFile(target, name string, content []byte) error {
	ffn := swag.ToFileName(name)
	if !strings.HasSuffix(ffn, "_test") {
		ffn += "_test"
	}
	ffn += ".go"

	res, err := formatGoFile(filepath.Join(target, ffn), content)
	if err != nil {
		log.Println(err)
		return writeFile(target, ffn, content)
	}

	return writeFile(target, ffn, res)
}

func writeFile(target, ffn string, content []byte) error {
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(target, ffn), content, 0644)
}

func commentedLines(str string) string {
	lines := strings.Split(str, "\n")
	var commented []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			if !strings.HasPrefix(strings.TrimSpace(line), "//") {
				commented = append(commented, "// "+line)
			} else {
				commented = append(commented, line)
			}
		}
	}
	return strings.Join(commented, "\n")
}

func gatherModels(specDoc *spec.Document, modelNames []string) (map[string]spec.Schema, error) {
	models, mnc := make(map[string]spec.Schema), len(modelNames)
	defs := specDoc.Spec().Definitions

	if mnc > 0 {
		var unknownModels []string
		for _, m := range modelNames {
			_, ok := defs[m]
			if !ok {
				unknownModels = append(unknownModels, m)
			}
		}
		if len(unknownModels) != 0 {
			return nil, fmt.Errorf("unknown models: %s", strings.Join(unknownModels, ", "))
		}
	}
	for k, v := range defs {
		if mnc == 0 {
			models[k] = v
		}
		for _, nm := range modelNames {
			if k == nm {
				models[k] = v
			}
		}
	}
	return models, nil
}

func appNameOrDefault(specDoc *spec.Document, name, defaultName string) string {
	if strings.TrimSpace(name) == "" {
		if specDoc.Spec().Info != nil && strings.TrimSpace(specDoc.Spec().Info.Title) != "" {
			name = specDoc.Spec().Info.Title
		} else {
			name = defaultName
		}
	}
	return strings.TrimSuffix(swag.ToGoName(name), "API")
}

func containsString(names []string, name string) bool {
	for _, nm := range names {
		if nm == name {
			return true
		}
	}
	return false
}

type opRef struct {
	Method string
	Path   string
	Key    string
	ID     string
	Op     *spec.Operation
}

type opRefs []opRef

func (o opRefs) Len() int           { return len(o) }
func (o opRefs) Swap(i, j int)      { o[i], o[j] = o[j], o[i] }
func (o opRefs) Less(i, j int) bool { return o[i].Key < o[j].Key }

func gatherOperations(specDoc *spec.Document, operationIDs []string) map[string]opRef {
	var oprefs opRefs

	for method, pathItem := range specDoc.Operations() {
		for path, operation := range pathItem {
			// nm := ensureUniqueName(operation.ID, method, path, operations)
			vv := *operation
			oprefs = append(oprefs, opRef{
				Key:    swag.ToGoName(strings.ToLower(method) + " " + path),
				Method: method,
				Path:   path,
				ID:     vv.ID,
				Op:     &vv,
			})
		}
	}

	sort.Sort(oprefs)

	operations := make(map[string]opRef)
	for _, opr := range oprefs {
		nm := opr.ID
		if nm == "" {
			nm = opr.Key
		}

		_, found := operations[nm]
		if found {
			nm = opr.Key
		}
		if len(operationIDs) == 0 || containsString(operationIDs, opr.ID) || containsString(operationIDs, nm) {
			opr.ID = nm
			opr.Op.ID = nm
			operations[nm] = opr
		}
	}

	return operations
}

func pascalize(arg string) string {
	if len(arg) == 0 || arg[0] > '9' {
		return swag.ToGoName(arg)
	}

	return swag.ToGoName("Nr " + arg)
}
