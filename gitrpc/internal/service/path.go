// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"fmt"
	"path/filepath"
)

// getFullPathForRepo returns the full path of a repo given the root dir of repos and the uid of the repo.
// NOTE: Split repos into subfolders using their prefix to distribute repos accross a set of folders.
func getFullPathForRepo(reposRoot, uid string) string {
	// ASSUMPTION: repoUID is of lenth at least 4 - otherwise we have trouble either way.
	return filepath.Join(
		reposRoot, // root folder
		uid[0:2],  // first subfolder
		uid[2:4],  // second subfolder
		fmt.Sprintf("%s.%s", uid[4:], gitRepoSuffix), // remainder with .git
	)
}
