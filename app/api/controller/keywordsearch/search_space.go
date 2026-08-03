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

package keywordsearch

import (
	"context"
	"fmt"
	"strings"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
)

type SearchSpaceInput struct {
	Query     string
	Limit     int
	Regex     bool
	RepoPaths []string
	Recursive bool
}

func (c *Controller) SearchSpace(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	in SearchSpaceInput,
) (types.SearchResult, error) {
	var repoPaths []string
	var spacePaths []string
	var recursive bool

	// Resolve the space ref (which can be a path or a numeric ID) to its actual path,
	// so the containment check below compares real paths rather than raw refs.
	space, err := c.spaceFinder.FindByRef(ctx, spaceRef)
	if err != nil {
		return types.SearchResult{}, fmt.Errorf("failed to find space by ref %q: %w", spaceRef, err)
	}

	spacePath := space.Path

	if len(in.RepoPaths) > 0 {
		if in.Recursive {
			return types.SearchResult{},
				usererror.BadRequestf("Can't use recursive search and specify list of repositories.")
		}

		for _, repoPath := range in.RepoPaths {
			if !strings.HasPrefix(repoPath, spacePath+string(types.PathSeparator)) {
				return types.SearchResult{},
					usererror.BadRequestf("Repository %q must be in space %q.", repoPath, spacePath)
			}
		}

		repoPaths = in.RepoPaths
	} else {
		spacePaths = []string{spacePath}
		recursive = in.Recursive
	}

	return c.Search(ctx, session, types.SearchInput{
		Query:          in.Query,
		RepoPaths:      repoPaths,
		SpacePaths:     spacePaths,
		MaxResultCount: in.Limit,
		EnableRegex:    in.Regex,
		Recursive:      recursive,
	})
}
