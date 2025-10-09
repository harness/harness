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

package api

import (
	"context"
	"fmt"
	"io"
	"path"
)

type FileContent struct {
	Path    string
	Content []byte
}

//nolint:gocognit
func (g *Git) MatchFiles(
	ctx context.Context,
	repoPath string,
	rev string,
	treePath string,
	pattern string,
	maxSize int,
) ([]FileContent, error) {
	nodes, err := lsDirectory(ctx, repoPath, rev, treePath, false, false)
	if err != nil {
		return nil, fmt.Errorf("failed to list files in match files: %w", err)
	}

	catFileWriter, catFileReader, catFileStop := CatFileBatch(ctx, repoPath, nil)
	defer catFileStop()

	var files []FileContent
	for i := range nodes {
		if nodes[i].NodeType != TreeNodeTypeBlob {
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

		_, err = catFileWriter.Write([]byte(nodes[i].SHA.String() + "\n"))
		if err != nil {
			return nil, fmt.Errorf("failed to ask for file content from cat file batch: %w", err)
		}

		output, err := ReadBatchHeaderLine(catFileReader)
		if err != nil {
			return nil, fmt.Errorf("failed to read cat-file batch header: %w", err)
		}

		reader := io.LimitReader(catFileReader, output.Size+1) // plus eol

		if output.Size > int64(maxSize) {
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

		files = append(files, FileContent{
			Path:    nodes[i].Path,
			Content: data,
		})
	}

	_ = catFileWriter.Close()

	return files, nil
}
