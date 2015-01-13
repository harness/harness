package repo

import (
	"testing"
)

func TestIsRemote(t *testing.T) {
	repos := []struct {
		path   string
		remote bool
	}{
		{"git://github.com/foo/far", true},
		{"git://github.com/foo/far.git", true},
		{"git@github.com:foo/far", true},
		{"git@github.com:foo/far.git", true},
		{"http://github.com/foo/far.git", true},
		{"https://github.com/foo/far.git", true},
		{"ssh://baz.com/foo/far.git", true},
		{"https://bitbucket.org/foo/bar", true},
		{"ssh://hg@bitbucket.org/foo/bar", true},
		{"/var/lib/src", false},
		{"/home/ubuntu/src", false},
		{"src", false},
	}

	for _, r := range repos {
		repo := Repo{Path: r.path}
		if remote := repo.IsRemote(); remote != r.remote {
			t.Errorf("IsRemote %s was %v, expected %v", r.path, remote, r.remote)
		}
	}
}
