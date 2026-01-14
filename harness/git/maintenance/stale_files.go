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

package maintenance

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

const (
	deleteTempFilesOlderThanDuration = 24 * time.Hour
)

func FindTempObjects(repoPath string) ([]string, error) {
	var files []string

	objectDir := filepath.Join(repoPath, "objects")

	if err := filepath.WalkDir(objectDir, func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrPermission) || errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}

		// Git creates only temp objects, packfiles and packfile indices.
		if dirEntry.IsDir() {
			return nil
		}

		isStale, err := isStaleTempObject(dirEntry)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}

			return fmt.Errorf("checking for stale temporary object: %w", err)
		}

		if !isStale {
			return nil
		}

		files = append(files, path)

		return nil
	}); err != nil {
		return nil, fmt.Errorf("walking object directory: %w", err)
	}

	return files, nil
}

func isStaleTempObject(dirEntry fs.DirEntry) (bool, error) {
	if !strings.HasPrefix(dirEntry.Name(), "tmp_") {
		return false, nil
	}

	fi, err := dirEntry.Info()
	if err != nil {
		return false, err
	}

	if time.Since(fi.ModTime()) <= deleteTempFilesOlderThanDuration {
		return false, nil
	}

	return true, nil
}
