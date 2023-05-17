// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
