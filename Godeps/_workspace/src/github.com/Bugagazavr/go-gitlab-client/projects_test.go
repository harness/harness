package gogitlab

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestALlProjects(t *testing.T) {
	ts, gitlab := Stub("stubs/projects/index.json")
	projects, err := gitlab.AllProjects()

	assert.Equal(t, err, nil)
	assert.Equal(t, len(projects), 2)
	defer ts.Close()
}

func TestProjects(t *testing.T) {
	ts, gitlab := Stub("stubs/projects/index.json")
	projects, err := gitlab.Projects(1, 100)

	assert.Equal(t, err, nil)
	assert.Equal(t, len(projects), 2)
	defer ts.Close()
}

func TestProject(t *testing.T) {
	ts, gitlab := Stub("stubs/projects/show.json")
	project, err := gitlab.Project("1")

	assert.Equal(t, err, nil)
	assert.IsType(t, new(Project), project)
	assert.Equal(t, project.SshRepoUrl, "git@example.com:diaspora/diaspora-project-site.git")
	assert.Equal(t, project.HttpRepoUrl, "http://example.com/diaspora/diaspora-project-site.git")
	defer ts.Close()
}

func TestProjectBranches(t *testing.T) {
	ts, gitlab := Stub("stubs/projects/branches/index.json")
	branches, err := gitlab.ProjectBranches("1")

	assert.Equal(t, err, nil)
	assert.Equal(t, len(branches), 2)
	defer ts.Close()
}

func TestProjectMergeRequests(t *testing.T) {
	ts, gitlab := Stub("stubs/projects/merge_requests/index.json")
	defer ts.Close()
	mr, err := gitlab.ProjectMergeRequests("1", 0, 30, "all")

	assert.Equal(t, err, nil)
	assert.Equal(t, len(mr), 1)

	if len(mr) > 0 {
		assert.Equal(t, mr[0].TargetBranch, "master")
		assert.Equal(t, mr[0].SourceBranch, "test1")
	}
}

func TestSearchProjectId(t *testing.T) {
	ts, gitlab := Stub("stubs/projects/index.json")

	namespace := "Brightbox"
	name := "Puppet"
	id, err := gitlab.SearchProjectId(namespace, name)

	assert.Equal(t, err, nil)
	assert.Equal(t, id, 6)
	defer ts.Close()
}
