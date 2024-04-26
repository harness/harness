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
	"bufio"
	"context"
	"strings"
)

type Submodule struct {
	Name string
	URL  string
}

// GetSubmodule returns the submodule at the given path reachable from ref.
// Note: ref can be Branch / Tag / CommitSHA.
func (g *Git) GetSubmodule(
	ctx context.Context,
	repoPath string,
	ref string,
	treePath string,
) (*Submodule, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}
	treePath = cleanTreePath(treePath)

	// Get the commit object for the ref
	commit, err := g.GetFullCommitID(ctx, repoPath, ref)
	if err != nil {
		return nil, processGitErrorf(err, "error getting commit for ref '%s'", ref)
	}

	node, err := g.GetTreeNode(ctx, repoPath, commit.String(), ".gitmodules")
	if err != nil {
		return nil, processGitErrorf(err, "error reading  tree node for ref '%s' with commit '%s'",
			ref, commit)
	}

	reader, err := GetBlob(ctx, repoPath, nil, node.SHA, 0)
	if err != nil {
		return nil, processGitErrorf(err, "error reading  commit for ref '%s'", ref)
	}
	defer reader.Content.Close()

	modules, err := GetSubModules(reader)
	if err != nil {
		return nil, processGitErrorf(err, "error getting submodule '%s' from commit", treePath)
	}

	return modules[treePath], nil
}

// GetSubModules get all the sub modules of current revision git tree.
func GetSubModules(rd *BlobReader) (map[string]*Submodule, error) {
	var isModule bool
	var path string
	submodules := make(map[string]*Submodule, 4)
	scanner := bufio.NewScanner(rd.Content)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "[submodule") {
			isModule = true
			continue
		}
		if isModule {
			fields := strings.Split(scanner.Text(), "=")
			k := strings.TrimSpace(fields[0])
			if k == "path" {
				path = strings.TrimSpace(fields[1])
			} else if k == "url" {
				submodules[path] = &Submodule{path, strings.TrimSpace(fields[1])}
				isModule = false
			}
		}
	}

	return submodules, nil
}
