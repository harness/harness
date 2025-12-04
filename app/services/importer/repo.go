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

package importer

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func NewRepo(
	spaceID int64,
	spacePath string,
	identifier string,
	description string,
	principal *types.Principal,
	defaultBranch string,
) *types.Repository {
	now := time.Now().UnixMilli()
	gitTempUID := fmt.Sprintf("importing-%s-%d", hash(fmt.Sprintf("%d:%s", spaceID, identifier)), now)
	return &types.Repository{
		Version:       0,
		ParentID:      spaceID,
		Identifier:    identifier,
		GitUID:        gitTempUID, // the correct git UID will be set by the job handler
		Description:   description,
		CreatedBy:     principal.ID,
		Created:       now,
		Updated:       now,
		LastGITPush:   now, // even in case of an empty repo, the git repo got created.
		ForkID:        0,
		DefaultBranch: defaultBranch,
		State:         enum.RepoStateGitImport,
		Path:          paths.Concatenate(spacePath, identifier),
		Tags:          json.RawMessage(`{}`),
	}
}
