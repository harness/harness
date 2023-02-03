// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
	// Pre indicates prerelease.
	Pre = ""
	// Dev indicates development branch. Releases will be empty string.
	Dev string

	// Version is the specification version that the package types support.
	Version = semver.Version{
		Major:      parseVersionNumber(major),
		Minor:      parseVersionNumber(minor),
		Patch:      parseVersionNumber(patch),
		PreRelease: semver.PreRelease(Pre),
		Metadata:   Dev,
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
