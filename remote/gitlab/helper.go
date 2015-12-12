package gitlab

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/drone/drone/remote/gitlab/client"
)

// NewClient is a helper function that returns a new GitHub
// client using the provided OAuth token.
func NewClient(url, accessToken string, skipVerify bool) *client.Client {
	client := client.New(url, "/api/v3", accessToken, skipVerify)
	return client
}

// IsRead is a helper function that returns true if the
// user has Read-only access to the repository.
func IsRead(proj *client.Project) bool {
	var user = proj.Permissions.ProjectAccess
	var group = proj.Permissions.GroupAccess

	switch {
	case proj.Public:
		return true
	case user != nil && user.AccessLevel >= 20:
		return true
	case group != nil && group.AccessLevel >= 20:
		return true
	default:
		return false
	}
}

// IsWrite is a helper function that returns true if the
// user has Read-Write access to the repository.
func IsWrite(proj *client.Project) bool {
	var user = proj.Permissions.ProjectAccess
	var group = proj.Permissions.GroupAccess

	switch {
	case user != nil && user.AccessLevel >= 30:
		return true
	case group != nil && group.AccessLevel >= 30:
		return true
	default:
		return false
	}
}

// IsAdmin is a helper function that returns true if the
// user has Admin access to the repository.
func IsAdmin(proj *client.Project) bool {
	var user = proj.Permissions.ProjectAccess
	var group = proj.Permissions.GroupAccess

	switch {
	case user != nil && user.AccessLevel >= 40:
		return true
	case group != nil && group.AccessLevel >= 40:
		return true
	default:
		return false
	}
}

// GetKeyTitle is a helper function that generates a title for the
// RSA public key based on the username and domain name.
func GetKeyTitle(rawurl string) (string, error) {
	var uri, err = url.Parse(rawurl)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("drone@%s", uri.Host), nil
}

func ns(owner, name string) string {
	return fmt.Sprintf("%s%%2F%s", owner, name)
}

func GetUserEmail(c *client.Client, defaultURL string) (*client.Client, error) {
	return c, nil
}

func GetProjectId(r *Gitlab, c *client.Client, owner, name string) (projectId string, err error) {
	if r.Search {
		_projectId, err := c.SearchProjectId(owner, name)
		if err != nil || _projectId == 0 {
			return "", err
		}
		projectId := strconv.Itoa(_projectId)
		return projectId, nil
	} else {
		projectId := ns(owner, name)
		return projectId, nil
	}
}
