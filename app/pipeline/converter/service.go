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

	"github.com/harness/gitness/app/pipeline/file"
	"github.com/harness/gitness/types"
)

type (
	// ConvertArgs represents a request to the pipeline
	// conversion service.
	ConvertArgs struct {
		Repo         *types.Repository `json:"repository,omitempty"`
		RepoIsPublic bool              `json:"repo_is_public,omitempty"`
		Pipeline     *types.Pipeline   `json:"pipeline,omitempty"`
		Execution    *types.Execution  `json:"execution,omitempty"`
		File         *file.File        `json:"config,omitempty"`
	}

	// Service converts a file which is in starlark/jsonnet form by looking
	// at the extension and calling the appropriate parser.
	Service interface {
		Convert(ctx context.Context, args *ConvertArgs) (*file.File, error)
	}
)
