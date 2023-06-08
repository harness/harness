// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import "regexp"

var matchCommitSHA = regexp.MustCompile("^[0-9a-f]+$")

func ValidateCommitSHA(commitSHA string) bool {
	if len(commitSHA) != 40 && len(commitSHA) != 64 {
		return false
	}

	return matchCommitSHA.MatchString(commitSHA)
}
