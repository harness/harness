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

package config

import (
	"context"

	"github.com/drone/drone/core"
)

// Repository returns a configuration service that fetches the yaml
// directly from the source code management (scm) system.
func Repository(service core.FileService) core.ConfigService {
	return &repo{files: service}
}

type repo struct {
	files core.FileService
}

func (r *repo) Find(ctx context.Context, req *core.ConfigArgs) (*core.Config, error) {
	raw, err := r.files.Find(ctx, req.User, req.Repo.Slug, req.Build.After, req.Build.Ref, req.Repo.Config)
	if err != nil {
		return nil, err
	}
	return &core.Config{
		Data: string(raw.Data),
	}, err
}
