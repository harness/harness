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

package git

import (
	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/git/adapter"
	"github.com/harness/gitness/git/storage"
	gittypes "github.com/harness/gitness/git/types"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideGITAdapter,
	ProvideService,
)

func ProvideGITAdapter(
	repoProvider *adapter.GoGitRepoProvider,
	lastCommitCache cache.Cache[adapter.CommitEntryKey, *gittypes.Commit],
) (Adapter, error) {
	return adapter.New(repoProvider, lastCommitCache)
}

func ProvideService(config *types.Config, adapter Adapter, storage storage.Store) (Interface, error) {
	return New(
		config.Git.Root,
		config.Git.TmpDir,
		adapter,
		storage,
		config.Git.HookPath,
	)
}
