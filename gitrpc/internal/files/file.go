// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package files

import (
	"path"
	"strings"
)

// CleanUploadFileName Trims a filename and returns empty string if it is a .git directory.
func CleanUploadFileName(name string) string {
	// Rebase the filename
	name = strings.Trim(name, "\n")
	name = strings.Trim(path.Clean("/"+name), "/")
	// Git disallows any filenames to have a .git directory in them.
	for _, part := range strings.Split(name, "/") {
		if strings.ToLower(part) == ".git" {
			return ""
		}
	}
	return name
}
