// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"text/template"
)

const (
	ScriptTemplatesDir = "script_templates"
)

var (
	//go:embed  script_templates/*
	files           embed.FS
	scriptTemplates map[string]*template.Template
)

func init() {
	err := LoadTemplates()
	if err != nil {
		panic(fmt.Sprintf("error loading script templates: %v", err))
	}
}

func LoadTemplates() error {
	scriptTemplates = make(map[string]*template.Template)

	tmplFiles, err := fs.ReadDir(files, ScriptTemplatesDir)
	if err != nil {
		return fmt.Errorf("error reading script templates: %w", err)
	}

	for _, tmpl := range tmplFiles {
		if tmpl.IsDir() {
			continue
		}

		textTemplate, parsingErr := template.ParseFS(files, path.Join(ScriptTemplatesDir, tmpl.Name()))
		if parsingErr != nil {
			return fmt.Errorf("error parsing template %s: %w", tmpl.Name(), parsingErr)
		}

		scriptTemplates[tmpl.Name()] = textTemplate
	}

	return nil
}

func GenerateScriptFromTemplate(name string, data interface{}) (string, error) {
	if scriptTemplates[name] == nil {
		return "", fmt.Errorf("no script template found for %s", name)
	}

	tmplOutput := bytes.Buffer{}
	err := scriptTemplates[name].Execute(&tmplOutput, data)
	if err != nil {
		return "", fmt.Errorf("error executing template %s with data %+v: %w", name, data, err)
	}

	return tmplOutput.String(), nil
}
