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

package sharedrepo

import (
	"github.com/harness/gitness/git/directory"
)

const (
	repoSharedRepoSubdirName = "shared_temp"
)

type RunnerFactory interface {
	// Runner returns Runner that applies changes made in sharedrepo.SharedRepo to the original repository.
	Runner(repoUID string, repoPath string, alternates ...string) Runner

	// ReadOnlyRunner returns Runner that discards any changes made in sharedrepo.SharedRepo.
	ReadOnlyRunner(repoUID string, repoPath string, alternates ...string) ReadOnlyRunner

	// RefRunner returns RefRunner that changes only references of the repository.
	RefRunner(repoUID string, repoPath string) RefRunner

	// Directory return the location where ephemeral shared repositories would be created.
	Directory() string
}

type runnerFactory struct {
	sharedRepoRoot string
}

func NewRunnerFactory(
	repositoryRootDirectory string,
) (RunnerFactory, error) {
	sharedRepoRoot, err := directory.CreateSubdir(repositoryRootDirectory, repoSharedRepoSubdirName)
	if err != nil {
		return nil, err
	}

	return runnerFactory{
		sharedRepoRoot: sharedRepoRoot,
	}, nil
}

func (f runnerFactory) Runner(repoUID, repoPath string, alternates ...string) Runner {
	return defaultRunner{
		tmpDir:     f.sharedRepoRoot,
		repoUID:    repoUID,
		repoPath:   repoPath,
		alternates: alternates,
	}
}

func (f runnerFactory) ReadOnlyRunner(repoUID, repoPath string, alternates ...string) ReadOnlyRunner {
	return defaultReadOnlyRunner{
		tmpDir:     f.sharedRepoRoot,
		repoUID:    repoUID,
		repoPath:   repoPath,
		alternates: alternates,
	}
}

func (runnerFactory) RefRunner(repoUID, repoPath string) RefRunner {
	return defaultRefRunner{
		repoUID:  repoUID,
		repoPath: repoPath,
	}
}

func (f runnerFactory) Directory() string {
	return f.sharedRepoRoot
}
