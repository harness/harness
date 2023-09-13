// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
