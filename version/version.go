// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package version provides the version number.
package version

import "github.com/coreos/go-semver/semver"

var (
	// GitRepository is the git repository that was compiled.
	GitRepository string
	// GitCommit is the git commit that was compiled.
	GitCommit string
	// Major is for an API incompatible changes.
	Major int64 = 1
	// Minor is for functionality in a backwards-compatible manner.
	Minor int64
	// Patch is for backwards-compatible bug fixes.
	Patch int64
	// Pre indicates prerelease.
	Pre = ""
	// Dev indicates development branch. Releases will be empty string.
	Dev string
)

// Version is the specification version that the package types support.
var Version = semver.Version{
	Major:      Major,
	Minor:      Minor,
	Patch:      Patch,
	PreRelease: semver.PreRelease(Pre),
	Metadata:   Dev,
}
