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

// Package version provides the version number.
package version

import (
	"strconv"

	"github.com/coreos/go-semver/semver"
)

var (
	// GitRepository is the git repository that was compiled.
	GitRepository string
	// GitCommit is the git commit that was compiled.
	GitCommit string
)

var (
	// major is for an API incompatible changes.
	major string
	// minor is for functionality in a backwards-compatible manner.
	minor string
	// patch is for backwards-compatible bug fixes.
	patch string
	// pre indicates prerelease.
	pre = ""
	// dev indicates development branch. Releases will be empty string.
	dev string

	// Version is the specification version that the package types support.
	Version = semver.Version{
		Major:      parseVersionNumber(major),
		Minor:      parseVersionNumber(minor),
		Patch:      parseVersionNumber(patch),
		PreRelease: semver.PreRelease(pre),
		Metadata:   dev,
	}
)

func parseVersionNumber(versionNum string) int64 {
	if versionNum == "" {
		return 0
	}
	i, err := strconv.ParseInt(versionNum, 10, 64)
	if err != nil {
		panic(err)
	}
	return i
}
