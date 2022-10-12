// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"errors"
	"os"
	"path/filepath"
)

func getRepoRoot() string {
	repoRoot := os.Getenv("GITNESS_REPO_ROOT")
	if repoRoot == "" {
		homedir, err := os.UserHomeDir()
		if err == nil {
			repoRoot = homedir
		}
	}
	targetPath := filepath.Join(repoRoot, "repos")
	if _, err := os.Stat(targetPath); errors.Is(err, os.ErrNotExist) {
		if err = os.MkdirAll(targetPath, 0o700); err != nil {
			return repoRoot
		}
	}
	return targetPath
}
