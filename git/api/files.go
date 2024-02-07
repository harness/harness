package api

import (
	"path"
	"strings"
)

// CleanUploadFileName Trims a filename and returns empty string if it is a .git directory
func CleanUploadFileName(name string) string {
	// Rebase the filename
	name = strings.Trim(path.Clean("/"+name), "/")
	// Git disallows any filenames to have a .git directory in them.
	for _, part := range strings.Split(name, "/") {
		if strings.ToLower(part) == ".git" {
			return ""
		}
	}
	return name
}
