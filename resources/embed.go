// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package resources

import "embed"

var (
	//go:embed gitignore
	Gitignore embed.FS

	//go:embed license
	Licence embed.FS
)
