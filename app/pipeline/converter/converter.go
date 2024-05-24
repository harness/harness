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

package converter

import (
	"context"
	"strings"

	"github.com/harness/gitness/app/pipeline/converter/jsonnet"
	"github.com/harness/gitness/app/pipeline/converter/starlark"
	"github.com/harness/gitness/app/pipeline/file"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/types/enum"
)

const (
	jsonnetImportLimit = 1000
	starlarkStepLimit  = 50000
	starlarkSizeLimit  = 1000000
)

type converter struct {
	fileService  file.Service
	publicAccess publicaccess.Service
}

func newConverter(fileService file.Service, publicAccess publicaccess.Service) Service {
	return &converter{
		fileService:  fileService,
		publicAccess: publicAccess,
	}
}

func (c *converter) Convert(ctx context.Context, args *ConvertArgs) (*file.File, error) {
	path := args.Pipeline.ConfigPath

	// get public access visibility of the repo
	repoIsPublic, err := c.publicAccess.Get(ctx, enum.PublicResourceTypeRepo, args.Repo.Path)
	if err != nil {
		return nil, err
	}

	if isJSONNet(path) {
		str, err := jsonnet.Parse(
			args.Repo,
			repoIsPublic,
			args.Pipeline,
			args.Execution,
			args.File,
			c.fileService,
			jsonnetImportLimit,
		)
		if err != nil {
			return nil, err
		}
		return &file.File{Data: []byte(str)}, nil
	} else if isStarlark(path) {
		str, err := starlark.Parse(
			args.Repo,
			repoIsPublic,
			args.Pipeline,
			args.Execution,
			args.File,
			starlarkStepLimit,
			starlarkSizeLimit,
		)
		if err != nil {
			return nil, err
		}
		return &file.File{Data: []byte(str)}, nil
	}
	return args.File, nil
}

func isJSONNet(path string) bool {
	return strings.HasSuffix(path, ".drone.jsonnet")
}

func isStarlark(path string) bool {
	return strings.HasSuffix(path, ".drone.script") ||
		strings.HasSuffix(path, ".drone.star") ||
		strings.HasSuffix(path, ".drone.starlark")
}
