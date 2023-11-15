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

package adapter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"

	"github.com/harness/gitness/git/types"

	gogitobject "github.com/go-git/go-git/v5/plumbing/object"
)

// nolint:gocognit
func (a Adapter) MatchFiles(
	ctx context.Context,
	repoPath string,
	ref string,
	dirPath string,
	pattern string,
	maxSize int,
) ([]types.FileContent, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}
	_, refCommit, err := a.getGoGitCommit(ctx, repoPath, ref)
	if err != nil {
		return nil, err
	}

	tree, err := refCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree for the commit: %w", err)
	}

	if dirPath != "" {
		tree, err = tree.Tree(dirPath)
		if errors.Is(err, gogitobject.ErrDirectoryNotFound) {
			return nil, &types.PathNotFoundError{Path: dirPath}
		}
		if err != nil {
			return nil, fmt.Errorf("failed to navigate to %s directory: %w", dirPath, err)
		}
	}

	var files []types.FileContent
	for i := range tree.Entries {
		fileEntry := tree.Entries[i]
		ok, err := path.Match(pattern, fileEntry.Name)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		name := fileEntry.Name

		f, err := tree.TreeEntryFile(&fileEntry)
		if err != nil {
			return nil, fmt.Errorf("failed to get tree entry file %s: %w", name, err)
		}

		reader, err := f.Reader()
		if err != nil {
			return nil, fmt.Errorf("failed to open tree entry file %s: %w", name, err)
		}

		filePath := path.Join(dirPath, name)

		content, err := func(r io.ReadCloser) ([]byte, error) {
			defer func() {
				_ = r.Close()
			}()
			return io.ReadAll(io.LimitReader(reader, int64(maxSize)))
		}(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to read file content %s: %w", name, err)
		}

		if len(content) == maxSize {
			// skip truncated files
			continue
		}

		files = append(files, types.FileContent{
			Path:    filePath,
			Content: content,
		})
	}

	return files, nil
}
