package repo

import (
	"testing"
)

func TestIsRemote(t *testing.T) {
	repos := []struct {
		scm    string
		remote bool
	}{
		{"git", true},
		{"mercurial", true},
		{"mercurial", true},
		{"mercurial", true},
		{"git", true},
		{"git", true},
		{"mercurial", true},
		{"local", false},
		{"local", false},
		{"local", false},
	}

	for _, r := range repos {
		repo := Repo{Scm: r.scm}
		if remote := repo.IsRemote(); remote != r.remote {
			t.Errorf("IsRemote %s was %v, expected %v", r.scm, remote, r.remote)
		}
	}
}

func TestIsLocal(t *testing.T) {
	repos := []struct {
		scm   string
		local bool
	}{
		{"git", false},
		{"mercurial", false},
		{"mercurial", false},
		{"mercurial", false},
		{"git", false},
		{"git", false},
		{"mercurial", false},
		{"local", true},
		{"local", true},
		{"local", true},
	}

	for _, r := range repos {
		repo := Repo{Scm: r.scm}
		if local := repo.IsLocal(); local != r.local {
			t.Errorf("IsRemote %s was %v, expected %v", r.scm, local, r.local)
		}
	}
}
