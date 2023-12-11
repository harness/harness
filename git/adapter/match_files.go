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
	"fmt"
	"io"
	"path"

	"github.com/harness/gitness/git/types"

	gitea "code.gitea.io/gitea/modules/git"
)

//nolint:gocognit
func (a Adapter) MatchFiles(
	ctx context.Context,
	repoPath string,
	rev string,
	treePath string,
	pattern string,
	maxSize int,
) ([]types.FileContent, error) {
	nodes, err := lsDirectory(ctx, repoPath, rev, treePath)
	if err != nil {
		return nil, fmt.Errorf("failed to list files in match files: %w", err)
	}

	catFileWriter, catFileReader, catFileStop := gitea.CatFileBatch(ctx, repoPath)
	defer catFileStop()

	var files []types.FileContent
	for i := range nodes {
		if nodes[i].NodeType != types.TreeNodeTypeBlob {
			continue
		}

		fileName := nodes[i].Name
		ok, err := path.Match(pattern, fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to match file name against pattern: %w", err)
		}
		if !ok {
			continue
		}

		_, err = catFileWriter.Write([]byte(nodes[i].Sha + "\n"))
		if err != nil {
			return nil, fmt.Errorf("failed to ask for file content from cat file batch: %w", err)
		}

		_, _, size, err := gitea.ReadBatchLine(catFileReader)
		if err != nil {
			return nil, fmt.Errorf("failed to read cat-file batch header: %w", err)
		}

		reader := io.LimitReader(catFileReader, size+1) // plus eol

		if size > int64(maxSize) {
			_, err = io.Copy(io.Discard, reader)
			if err != nil {
				return nil, fmt.Errorf("failed to discard a large file: %w", err)
			}
		}

		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to read cat-file content: %w", err)
		}

		if len(data) > 0 {
			data = data[:len(data)-1]
		}

		if len(data) == 0 {
			continue
		}

		files = append(files, types.FileContent{
			Path:    nodes[i].Path,
			Content: data,
		})
	}

	_ = catFileWriter.Close()

	return files, nil
}
