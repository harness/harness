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

func TestIsGit(t *testing.T) {
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
		{"svn://gcc.gnu.org/svn/gcc/branches/gccgo", false},
		{"https://code.google.com/p/go", false},
	}

	for _, r := range repos {
		repo := Repo{Path: r.path}
		if remote := repo.IsGit(); remote != r.remote {
			t.Errorf("IsGit %s was %v, expected %v", r.path, remote, r.remote)
		}
	}
}

func TestIsTrusted(t *testing.T) {
	repos := []struct {
		private bool
		PR      string
		trusted bool
	}{
		{true, "1", true},
		{false, "1", false},
		{true, "", true},
		{false, "", true},
	}

	for _, r := range repos {
		repo := Repo{Private: r.private, PR: r.PR}
		if trusted := repo.IsTrusted(); trusted != r.trusted {
			t.Errorf("IsTrusted was %v, expected %v", trusted, r.trusted)
		}
	}
}
