// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitea

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"

	"github.com/harness/gitness/gitrpc/internal/types"

	gogitobject "github.com/go-git/go-git/v5/plumbing/object"
)

func (g Adapter) MatchFiles(ctx context.Context,
	repoPath string,
	ref string,
	dirPath string,
	pattern string,
	maxSize int,
) ([]types.FileContent, error) {
	_, refCommit, err := g.getGoGitCommit(ctx, repoPath, ref)
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
	for _, fileEntry := range tree.Entries {
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
