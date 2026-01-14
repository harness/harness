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

package resolver

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types/enum"

	v1yaml "github.com/drone/spec/dist/go"
	"github.com/drone/spec/dist/go/parse"
)

// Resolve returns a resolve function which resolves plugins and templates.
// It searches for plugins globally and for templates in the same space and substitutes
// them in the pipeline yaml.
func Resolve(
	ctx context.Context,
	pluginStore store.PluginStore,
	templateStore store.TemplateStore,
	spaceID int64,
) func(name, kind, typ, version string) (*v1yaml.Config, error) {
	return func(name, kind, typ, version string) (*v1yaml.Config, error) {
		k, err := enum.ParseResolverKind(kind)
		if err != nil {
			return nil, err
		}
		t, err := enum.ParseResolverType(typ)
		if err != nil {
			return nil, err
		}
		if k == enum.ResolverKindPlugin && t != enum.ResolverTypeStep {
			return nil, fmt.Errorf("only step level plugins are currently supported")
		}
		if k == enum.ResolverKindPlugin {
			plugin, err := pluginStore.Find(ctx, name, version)
			if err != nil {
				return nil, fmt.Errorf("could not lookup plugin: %w", err)
			}
			// Convert plugin to v1yaml spec
			config, err := parse.ParseString(plugin.Spec)
			if err != nil {
				return nil, fmt.Errorf("could not unmarshal plugin to v1yaml spec: %w", err)
			}
			return config, nil
		}

		// Search for templates in the space
		template, err := templateStore.FindByIdentifierAndType(ctx, spaceID, name, t)
		if err != nil {
			return nil, fmt.Errorf("could not find template: %w", err)
		}

		// Try to parse the template into v1 yaml
		config, err := parse.ParseString(template.Data)
		if err != nil {
			return nil, fmt.Errorf("could not unmarshal template to v1yaml spec: %w", err)
		}

		return config, nil
	}
}
