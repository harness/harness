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

package validator

import (
	"context"
	"path/filepath"

	"github.com/drone/drone/core"
)

// Filter returns a validation service that skips
// pipelines that do not match the filter criteria.
func Filter(include, exclude []string) core.ValidateService {
	return &filter{
		include: include,
		exclude: exclude,
	}
}

type filter struct {
	include []string
	exclude []string
}

func (f *filter) Validate(ctx context.Context, in *core.ValidateArgs) error {
	if len(f.include) > 0 {
		for _, pattern := range f.include {
			ok, _ := filepath.Match(pattern, in.Repo.Slug)
			if ok {
				return nil
			}
		}

		// if the include list is specified, and the
		// repository does not match any patterns in
		// the include list, it should be skipped.
		return core.ErrValidatorSkip
	}

	if len(f.exclude) > 0 {
		for _, pattern := range f.exclude {
			ok, _ := filepath.Match(pattern, in.Repo.Slug)
			if ok {
				// if the exclude list is specified, and
				// the repository matches a pattern in the
				// exclude list, it should be skipped.
				return core.ErrValidatorSkip
			}
		}
	}

	return nil
}
