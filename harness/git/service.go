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
	"fmt"
	"os"
	"path/filepath"

	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/storage"
	"github.com/harness/gitness/git/types"
)

const (
	repoSubdirName           = "repos"
	repoSharedRepoSubdirName = "shared_temp"
	ReposGraveyardSubdirName = "cleanup"
)

type Service struct {
	reposRoot         string
	sharedRepoRoot    string
	git               *api.Git
	hookClientFactory hook.ClientFactory
	store             storage.Store
	gitHookPath       string
	reposGraveyard    string
}

func New(
	config types.Config,
	adapter *api.Git,
	hookClientFactory hook.ClientFactory,
	storage storage.Store,
) (*Service, error) {
	// Create repos folder
	reposRoot, err := createSubdir(config.Root, repoSubdirName)
	if err != nil {
		return nil, err
	}

	// create a temp dir for deleted repositories
	// this dir should get cleaned up peridocally if it's not empty
	reposGraveyard, err := createSubdir(config.Root, ReposGraveyardSubdirName)
	if err != nil {
		return nil, err
	}

	sharedRepoDir, err := createSubdir(config.Root, repoSharedRepoSubdirName)
	if err != nil {
		return nil, err
	}

	return &Service{
		reposRoot:         reposRoot,
		sharedRepoRoot:    sharedRepoDir,
		reposGraveyard:    reposGraveyard,
		git:               adapter,
		hookClientFactory: hookClientFactory,
		store:             storage,
		gitHookPath:       config.HookPath,
	}, nil
}

func createSubdir(root, subdir string) (string, error) {
	subdirPath := filepath.Join(root, subdir)

	err := os.MkdirAll(subdirPath, fileMode700)
	if err != nil {
		return "", fmt.Errorf("failed to create directory, path=%s: %w", subdirPath, err)
	}

	return subdirPath, nil
}
