package cron

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
)

func TestCleanupRepoGraveyardFunc(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	//create a dummy repository
	testRepo, _ := ioutil.TempDir(tmpDir, "TestRepo100")
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
