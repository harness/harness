package vcsutil

import (
	"testing"
)

func TestIsGit(t *testing.T) {
	repos := []struct {
		path string
		git  bool
	}{
		{"git://github.com/foo/far", true},
		{"git://github.com/foo/far.git", true},
		{"git@github.com:foo/far", true},
		{"git@github.com:foo/far.git", true},
		{"http://github.com/foo/far.git", true},
		{"https://github.com/foo/far.git", true},
		{"ssh://baz.com/foo/far.git", true},
		{"svn://gcc.gnu.org/svn/gcc/branches/gccgo", false},
		{"https://code.google.com/p/go", false},
	}

	for _, r := range repos {
		if git := IsGit(r.path); git != r.git {
			t.Errorf("IsGit %s was %v, expected %v", r.path, git, r.git)
		}
	}
}
