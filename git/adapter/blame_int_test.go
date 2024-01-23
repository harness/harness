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

package adapter_test

import (
	"context"
	"io"
	"testing"

	"github.com/harness/gitness/errors"
)

func TestBlameEmptyFile(t *testing.T) {
	git := setupGit(t)
	repo, teardown := setupRepo(t, git, "testblameemptyfile")
	defer teardown()

	baseBranch := "main"
	// write empty file to main branch
	_, parentSHA := writeFile(t, repo, "file.txt", "", nil)

	err := repo.SetReference("refs/heads/"+baseBranch, parentSHA.String())
	if err != nil {
		t.Fatalf("failed updating reference '%s': %v", baseBranch, err)
	}

	reader := git.Blame(context.Background(), repo.Path, "main", "file.txt", 0, 0)

	part, err := reader.NextPart()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return
		}
		t.Errorf("Blame reader should return empty string but got error: %v", err)
		return
	}

	if part != nil {
		t.Errorf("Blame reader should be nil but got: %v", part)
		return
	}
}
