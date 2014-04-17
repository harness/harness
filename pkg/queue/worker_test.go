package queue

import (
	. "github.com/drone/drone/pkg/model"
	"testing"
)

func Test_getBuildUrl(t *testing.T) {
	repo := &Repo{
		Slug: "examplegit.com/owner/repo",
	}
	commit := &Commit{
		Hash:   "abc",
		Branch: "example",
	}

	expected := "http://examplehost.com/examplegit.com/owner/repo/commit/abc?branch=example"
	output := getBuildUrl("http://examplehost.com", repo, commit)

	if output != expected {
		t.Errorf("Failed to build url. Expected: %s, got %s", expected, output)
	}

	commit.Branch = "url/unsafe/branch"
	expected = "http://examplehost.com/examplegit.com/owner/repo/commit/abc?branch=url%2Funsafe%2Fbranch"
	output = getBuildUrl("http://examplehost.com", repo, commit)

	if output != expected {
		t.Errorf("Failed to build url. Expected: %s, got %s", expected, output)
	}
}
