// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package tempdir

import (
	"fmt"
	"os"
)

// CreateTemporaryPath creates a temporary path.
func CreateTemporaryPath(reposTempPath, prefix string) (string, error) {
	if reposTempPath != "" {
		if err := os.MkdirAll(reposTempPath, os.ModePerm); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", reposTempPath, err)
		}
	}
	basePath, err := os.MkdirTemp(reposTempPath, prefix+".git")
	if err != nil {
		return "", fmt.Errorf("failed to create dir %s-*.git: %w", prefix, err)
	}
	return basePath, nil
}

// RemoveTemporaryPath removes the temporary path.
func RemoveTemporaryPath(basePath string) error {
	if _, err := os.Stat(basePath); !os.IsNotExist(err) {
		return os.RemoveAll(basePath)
	}
	return nil
}
