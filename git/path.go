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
	"path/filepath"
)

const (
	gitRepoSuffix = "git"
)

// getFullPathForRepo returns the full path of a repo given the root dir of repos and the uid of the repo.
// NOTE: Split repos into subfolders using their prefix to distribute repos across a set of folders.
func getFullPathForRepo(reposRoot, uid string) string {
	// ASSUMPTION: repoUID is of lenth at least 4 - otherwise we have trouble either way.
	return filepath.Join(
		reposRoot, // root folder
		uid[0:2],  // first subfolder
		uid[2:4],  // second subfolder
		fmt.Sprintf("%s.%s", uid[4:], gitRepoSuffix), // remainder with .git
	)
}
