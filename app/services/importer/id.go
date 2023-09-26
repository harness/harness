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
	"strconv"
	"strings"
)

const jobIDPrefix = "import-repo-"

func JobIDFromRepoID(repoID int64) string {
	return jobIDPrefix + strconv.FormatInt(repoID, 10)
}

func RepoIDFromJobID(jobID string) int64 {
	if !strings.HasPrefix(jobID, jobIDPrefix) {
		return 0
	}
	repoID, _ := strconv.ParseInt(jobID[len(jobIDPrefix):], 10, 64)
	return repoID
}
