package gogitlab

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRepoBranches(t *testing.T) {
	ts, gitlab := Stub("stubs/branches/index.json")
	branches, err := gitlab.RepoBranches("1")

	assert.Equal(t, err, nil)
	assert.Equal(t, len(branches), 1)
	defer ts.Close()
}

func TestRepoBranch(t *testing.T) {
	ts, gitlab := Stub("stubs/branches/show.json")
	branch, err := gitlab.RepoBranch("1", "master")

	assert.Equal(t, err, nil)
	assert.IsType(t, new(Branch), branch)
	assert.Equal(t, branch.Name, "master")
	defer ts.Close()
}

func TestRepoTags(t *testing.T) {
	ts, gitlab := Stub("stubs/tags/index.json")
	tags, err := gitlab.RepoTags("1")

	assert.Equal(t, err, nil)
	assert.Equal(t, len(tags), 1)
	defer ts.Close()
}

func TestRepoCommits(t *testing.T) {
	ts, gitlab := Stub("stubs/commits/index.json")
	commits, err := gitlab.RepoCommits("1")

	assert.Equal(t, err, nil)
	assert.Equal(t, len(commits), 2)
	defer ts.Close()
}

func TestRepoCommitComments(t *testing.T) {
	ts, gitlab := Stub("stubs/commits/comments/index.json")
	comments, err := gitlab.RepoCommitComments("1", "a9e6a5io4e695923c995ed2e836789b50oi77e0b")

	assert.Equal(t, err, nil)
	assert.Equal(t, len(comments), 1)
	defer ts.Close()
}
