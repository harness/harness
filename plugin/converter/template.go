// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build !oss

package converter

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"regexp"
	templating "text/template"

	"github.com/drone/drone/core"
	"github.com/drone/drone/plugin/converter/jsonnet"
	"github.com/drone/drone/plugin/converter/starlark"
	"github.com/drone/funcmap"

	"gopkg.in/yaml.v2"
)

var (
	// templateFileRE regex to verifying kind is template.
	templateFileRE              = regexp.MustCompilePOSIX("^kind:[[:space:]]+template[[:space:]]?+$")
	errTemplateNotFound         = errors.New("template converter: template name given not found")
	errTemplateSyntaxErrors     = errors.New("template converter: there is a problem with the yaml file provided")
	errTemplateExtensionInvalid = errors.New("template extension invalid. must be yaml, starlark or jsonnet")
)

func Template(templateStore core.TemplateStore, stepLimit uint64, sizeLimit uint64) core.ConvertService {
	return &templatePlugin{
		templateStore: templateStore,
		stepLimit: stepLimit,
		sizeLimit: sizeLimit,
	}
}

type templatePlugin struct {
	templateStore core.TemplateStore
	stepLimit uint64
	sizeLimit uint64
}

func (p *templatePlugin) Convert(ctx context.Context, req *core.ConvertArgs) (*core.Config, error) {
	// check type is yaml
	configExt := filepath.Ext(req.Repo.Config)

	if configExt != ".yml" && configExt != ".yaml" {
		return nil, nil
	}

	// check kind is template
	if templateFileRE.MatchString(req.Config.Data) == false {
		return nil, nil
	}
	// map to templateArgs
	var templateArgs core.TemplateArgs
	err := yaml.Unmarshal([]byte(req.Config.Data), &templateArgs)
	if err != nil {
		return nil, errTemplateSyntaxErrors
	}
	// get template from db
	template, err := p.templateStore.FindName(ctx, templateArgs.Load, req.Repo.Namespace)
	if err == sql.ErrNoRows {
		return nil, errTemplateNotFound
	}
	if err != nil {
		return nil, err
	}

	switch filepath.Ext(templateArgs.Load) {
	case ".yml", ".yaml":
		return parseYaml(req, template, templateArgs)
	case ".star", ".starlark", ".script":
		return parseStarlark(req, template, templateArgs, p.stepLimit, p.sizeLimit)
	case ".jsonnet":
		return parseJsonnet(req, template, templateArgs)
	default:
		return nil, errTemplateExtensionInvalid
	}
}

func parseYaml(req *core.ConvertArgs, template *core.Template, templateArgs core.TemplateArgs) (*core.Config, error) {
	data := map[string]interface{}{
		"build": toBuild(req.Build),
		"repo":  toRepo(req.Repo),
		"input": templateArgs.Data,
	}
	tmpl, err := templating.New(template.Name).Funcs(funcmap.SafeFuncs).Parse(template.Data)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	err = tmpl.Execute(&out, data)
	if err != nil {
		return nil, err
	}
	return &core.Config{
		Data: out.String(),
	}, nil
}

func parseJsonnet(req *core.ConvertArgs, template *core.Template, templateArgs core.TemplateArgs) (*core.Config, error) {
	file, err := jsonnet.Parse(req, nil, 0, template, templateArgs.Data)
	if err != nil {
		return nil, err
	}
	return &core.Config{
		Data: file,
	}, nil
}

func parseStarlark(req *core.ConvertArgs, template *core.Template, templateArgs core.TemplateArgs, stepLimit uint64, sizeLimit uint64) (*core.Config, error) {
	file, err := starlark.Parse(req, template, templateArgs.Data, stepLimit, sizeLimit)
	if err != nil {
		return nil, err
	}
	return &core.Config{
		Data: file,
	}, nil
}
