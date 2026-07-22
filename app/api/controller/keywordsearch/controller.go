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

	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/keywordsearch"
	"github.com/harness/gitness/types"
)

type Controller struct {
	authorizer       authz.Authorizer
	searcher         keywordsearch.Searcher
	repositoryFinder RepositoryFinder
	repositoryLister RepositoryLister
}

// RepositoryFinder is the interface that the keyword search controller uses to fetch a single repository by ref.
// The method should check if user has view permission to the repository.
// Currently, this is normally wired to repository controller.
type RepositoryFinder interface {
	Find(ctx context.Context, session *auth.Session, repoRef string) (*repo.RepositoryOutput, error)
}

// RepositoryLister is the interface that the keyword search controller uses to fetch a list repositories by space ref.
// The method should check if user has view permission to each of the repositories.
// Currently, this is normally wired to spaces controller.
type RepositoryLister interface {
	ListRepositories(
		ctx context.Context,
		session *auth.Session,
		spaceRef string,
		filter *types.RepoFilter,
	) ([]*repo.RepositoryOutput, int64, error)
}

func NewController(
	authorizer authz.Authorizer,
	searcher keywordsearch.Searcher,
	repositoryFinder RepositoryFinder,
	repositoryLister RepositoryLister,
) *Controller {
	return &Controller{
		authorizer:       authorizer,
		searcher:         searcher,
		repositoryFinder: repositoryFinder,
		repositoryLister: repositoryLister,
	}
}
