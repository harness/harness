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

package commit

import (
	"context"

	"github.com/harness/gitness/types"
)

type (
	// Service provides access to commit information via
	// the SCM provider. Today, this is Harness but it can
	// be extendible to any SCM provider.
	Service interface {
		// ref is the ref to fetch the commit from, eg refs/heads/master
		FindRef(ctx context.Context, repo *types.Repository, ref string) (*types.Commit, error)

		// FindCommit returns information about a commit in a repo.
		FindCommit(ctx context.Context, repo *types.Repository, sha string) (*types.Commit, error)
	}
)
