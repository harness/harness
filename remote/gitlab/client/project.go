package client

import (
	"encoding/json"
	"strconv"
	"strings"
)

const (
	searchUrl      = "/projects/search/:query"
	projectsUrl    = "/projects"
	projectUrl     = "/projects/:id"
	repoUrlRawFile = "/projects/:id/repository/blobs/:sha"
)

// Get a list of all projects owned by the authenticated user.
func (g *Client) AllProjects() ([]*Project, error) {
	var per_page = 100
	var projects []*Project

	for i := 1; true; i++ {
		contents, err := g.Projects(i, per_page)
		if err != nil {
			return projects, err
		}

		for _, value := range contents {
			projects = append(projects, value)
		}

		if len(projects) == 0 {
			break
		}

		if len(projects)/i < per_page {
			break
		}
	}

	return projects, nil
}

// Get a list of projects owned by the authenticated user.
func (c *Client) Projects(page int, per_page int) ([]*Project, error) {

	url, opaque := c.ResourceUrl(projectsUrl, nil, QMap{
		"page":     strconv.Itoa(page),
		"per_page": strconv.Itoa(per_page),
	})

	var projects []*Project

	contents, err := c.Do("GET", url, opaque, nil)
	if err == nil {
		err = json.Unmarshal(contents, &projects)
	}

	return projects, err
}

// Get a project by id
func (c *Client) Project(id string) (*Project, error) {
	url, opaque := c.ResourceUrl(projectUrl, QMap{":id": id}, nil)

	var project *Project

	contents, err := c.Do("GET", url, opaque, nil)
	if err == nil {
		err = json.Unmarshal(contents, &project)
	}

	return project, err
}

// Get Raw file content
func (c *Client) RepoRawFile(id, sha, filepath string) ([]byte, error) {
	url, opaque := c.ResourceUrl(
		repoUrlRawFile,
		QMap{
			":id":  id,
			":sha": sha,
		},
		QMap{
			"filepath": filepath,
		},
	)

	contents, err := c.Do("GET", url, opaque, nil)

	return contents, err
}

// Get a list of projects by query owned by the authenticated user.
func (c *Client) SearchProjectId(namespace string, name string) (id int, err error) {

	url, opaque := c.ResourceUrl(searchUrl, nil, QMap{
		":query": strings.ToLower(name),
	})

	var projects []*Project

	contents, err := c.Do("GET", url, opaque, nil)
	if err == nil {
		err = json.Unmarshal(contents, &projects)
	} else {
		return id, err
	}

	for _, project := range projects {
		if project.Namespace.Name == namespace && strings.ToLower(project.Name) == strings.ToLower(name) {
			id = project.Id
		}
	}

	return id, err
}
