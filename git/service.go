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
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/directory"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/sharedrepo"
	"github.com/harness/gitness/git/storage"
	"github.com/harness/gitness/git/types"
)

const (
	repoSubdirName           = "repos"
	ReposGraveyardSubdirName = "cleanup"
)

type Service struct {
	reposRoot           string
	git                 *api.Git
	hookClientFactory   hook.ClientFactory
	store               storage.Store
	sharedRunnerFactory sharedrepo.RunnerFactory
	gitHookPath         string
	reposGraveyard      string
}

func New(
	config types.Config,
	adapter *api.Git,
	hookClientFactory hook.ClientFactory,
	storage storage.Store,
	sharedRunnerFactory sharedrepo.RunnerFactory,
) (*Service, error) {
	// Create repos folder
	reposRoot, err := directory.CreateSubdir(config.Root, repoSubdirName)
	if err != nil {
		return nil, err
	}

	// create a temp dir for deleted repositories
	// this dir should get cleaned up peridocally if it's not empty
	reposGraveyard, err := directory.CreateSubdir(config.Root, ReposGraveyardSubdirName)
	if err != nil {
		return nil, err
	}

	return &Service{
		reposRoot:           reposRoot,
		reposGraveyard:      reposGraveyard,
		git:                 adapter,
		hookClientFactory:   hookClientFactory,
		store:               storage,
		sharedRunnerFactory: sharedRunnerFactory,
		gitHookPath:         config.HookPath,
	}, nil
}
