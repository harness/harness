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

package githook

import (
	"context"

	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/api"
)

// RestrictedGIT is a git client that is restricted to a subset of operations of git.Interface
// which can be executed on quarantine data that is part of git-hooks (e.g. pre-receive, update, ..)
// and don't alter the repo (so only read operations).
// NOTE: While it doesn't apply to all git-hooks (e.g. post-receive), we still use the interface across the board
// to "soft enforce" no write operations being executed as part of githooks.
type RestrictedGIT interface {
	IsAncestor(ctx context.Context, params git.IsAncestorParams) (git.IsAncestorOutput, error)
	ScanSecrets(ctx context.Context, param *git.ScanSecretsParams) (*git.ScanSecretsOutput, error)
	GetBranch(ctx context.Context, params *git.GetBranchParams) (*git.GetBranchOutput, error)
	Diff(ctx context.Context, in *git.DiffParams, files ...api.FileDiffRequest) (<-chan *git.FileDiff, <-chan error)
	GetBlob(ctx context.Context, params *git.GetBlobParams) (*git.GetBlobOutput, error)
	// TODO: remove. Kept for backwards compatibility.
	FindOversizeFiles(
		ctx context.Context,
		params *git.FindOversizeFilesParams,
	) (*git.FindOversizeFilesOutput, error)
}
