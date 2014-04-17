package notify

import (
	"github.com/drone/drone/pkg/model"
	"testing"
)

func Test_getBuildUrl(t *testing.T) {
	c := &Context{
		Host: "http://examplehost.com",
		Repo: &model.Repo{
			Slug: "examplegit.com/owner/repo",
		},
		Commit: &model.Commit{
			Hash:   "abc",
			Branch: "example",
		},
	}
	expected := "http://examplehost.com/examplegit.com/owner/repo/commit/abc?branch=example"
	output := getBuildUrl(c)

	if output != expected {
		t.Errorf("Failed to build url. Expected: %s, got %s", expected, output)
	}

	c.Commit.Branch = "url/unsafe/branch"
	expected = "http://examplehost.com/examplegit.com/owner/repo/commit/abc?branch=url%2Funsafe%2Fbranch"
	output = getBuildUrl(c)

	if output != expected {
		t.Errorf("Failed to build url. Expected: %s, got %s", expected, output)
	}

	c.Commit.Branch = ""
	expected = "http://examplehost.com/examplegit.com/owner/repo/commit/abc?"
	output = getBuildUrl(c)

	if output != expected {
		t.Errorf("Failed to build url. Expected: %s, got %s", expected, output)
	}
}
