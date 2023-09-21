// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cron

import (
	"context"
	"os"
	"testing"
)

func TestCleanupRepoGraveyardFunc(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	// create a dummy repository
	testRepo, _ := os.MkdirTemp(tmpDir, "TestRepo100")
	err := cleanupRepoGraveyard(ctx, tmpDir)
	if err != nil {
		t.Error("cleanupRepoGraveyard failed")
	}
	if _, err := os.Stat(testRepo); !os.IsNotExist(err) {
		t.Error("cleanupRepoGraveyard failed to remove the directory")
	}
}

func TestCleanupRepoGraveyardEmpty(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	err := cleanupRepoGraveyard(ctx, tmpDir)
	if err != nil {
		t.Error("cleanupRepoGraveyard failed")
	}
}
