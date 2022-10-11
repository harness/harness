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
