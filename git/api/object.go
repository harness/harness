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

import "fmt"

//// ObjectType git object type.
//type ObjectType string
//
//const (
//	// ObjectCommit commit object type.
//	ObjectCommit ObjectType = "commit"
//	// ObjectTree tree object type.
//	ObjectTree ObjectType = "tree"
//	// ObjectBlob blob object type.
//	ObjectBlob ObjectType = "blob"
//	// ObjectTag tag object type.
//	ObjectTag ObjectType = "tag"
//	// ObjectBranch branch object type.
//	ObjectBranch ObjectType = "branch"
//)

type GitObjectType string

const (
	GitObjectTypeCommit GitObjectType = "commit"
	GitObjectTypeTree   GitObjectType = "tree"
	GitObjectTypeBlob   GitObjectType = "blob"
	GitObjectTypeTag    GitObjectType = "tag"
)

func ParseGitObjectType(t string) (GitObjectType, error) {
	switch t {
	case string(GitObjectTypeCommit):
		return GitObjectTypeCommit, nil
	case string(GitObjectTypeBlob):
		return GitObjectTypeBlob, nil
	case string(GitObjectTypeTree):
		return GitObjectTypeTree, nil
	case string(GitObjectTypeTag):
		return GitObjectTypeTag, nil
	default:
		return GitObjectTypeBlob, fmt.Errorf("unknown git object type '%s'", t)
	}
}

type SortOrder int

const (
	SortOrderDefault SortOrder = iota
	SortOrderAsc
	SortOrderDesc
)
