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

// +build oss

package converter

import (
	"context"

	"github.com/drone/drone/core"
)

func Template(templateStore core.TemplateStore, stepLimit uint64, sizeLimit uint64) core.ConvertService {
	return &templatePlugin{
		templateStore: templateStore,
	}
}

type templatePlugin struct {
	templateStore core.TemplateStore
}

func (p *templatePlugin) Convert(ctx context.Context, req *core.ConvertArgs) (*core.Config, error) {
	return nil, nil
}
