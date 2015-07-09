package vcsutil

import (
	"strings"
)

func IsGit(path string) bool {
	switch {
	case strings.HasPrefix(path, "git://"):
		return true
	case strings.HasPrefix(path, "git@"):
		return true
	case strings.HasPrefix(path, "ssh://git@"):
		return true
	case strings.HasPrefix(path, "gitlab@"):
		return true
	case strings.HasPrefix(path, "ssh://gitlab@"):
		return true
	case strings.HasPrefix(path, "https://github"):
		return true
	case strings.HasPrefix(path, "http://github"):
		return true
	case strings.HasSuffix(path, ".git"):
		return true
	}

	// we could also ping the repository to check

	return false
}
