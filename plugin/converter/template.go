package converter

import (
	"context"
	"errors"
	"github.com/drone/drone/core"

	"github.com/drone/drone/plugin/converter/parser"

	"gopkg.in/yaml.v2"
	"regexp"
	"strings"
)

var (
	// templateFileRE regex to verifying kind is template.
	templateFileRE          = regexp.MustCompile("^kind:\\s+template+\\n")
	ErrTemplateNotFound     = errors.New("template converter: template name given not found")
	ErrTemplateSyntaxErrors = errors.New("template converter: there is a problem with the yaml file provided")
)

func Template(templateStore core.TemplateStore) core.ConvertService {
	return &templatePlugin{
		templateStore: templateStore,
	}
}

type templatePlugin struct {
	templateStore core.TemplateStore
}

func (p *templatePlugin) Convert(ctx context.Context, req *core.ConvertArgs) (*core.Config, error) {
	// check type is yaml
	if strings.HasSuffix(req.Repo.Config, ".yml") == false {
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
		return nil, ErrTemplateSyntaxErrors
	}
	// get template from db
	template, err := p.templateStore.FindName(ctx, templateArgs.Load)
	if err != nil {
		return nil, nil
	}
	if template == nil {
		return nil, ErrTemplateNotFound
	}
	// Check if file is Starlark
	if strings.HasSuffix(templateArgs.Load, ".script") ||
		strings.HasSuffix(templateArgs.Load, ".star") ||
		strings.HasSuffix(templateArgs.Load, ".starlark") {

		file, err := parser.ParseStarlark(req, template, templateArgs.Data)
		if err != nil {
			return nil, err
		}
		return &core.Config{
			Data: *file,
		}, nil
	}
	return nil, nil
}
