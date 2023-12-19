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

package adapter

import (
	"context"

	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/types"

	gitea "code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/setting"
)

type Adapter struct {
	traceGit        bool
	lastCommitCache cache.Cache[CommitEntryKey, *types.Commit]
	githookFactory  hook.ClientFactory
}

func New(
	config types.Config,
	lastCommitCache cache.Cache[CommitEntryKey, *types.Commit],
	githookFactory hook.ClientFactory,
) (Adapter, error) {
	// TODO: should be subdir of gitRoot? What is it being used for?
	setting.Git.HomePath = "home"

	err := gitea.InitSimple(context.Background())
	if err != nil {
		return Adapter{}, err
	}

	return Adapter{
		traceGit:        config.Trace,
		lastCommitCache: lastCommitCache,
		githookFactory:  githookFactory,
	}, nil
}
