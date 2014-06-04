package gitlab

import (
	"github.com/drone/drone/shared/remote"
)

type Client struct {
	config *Gitlab
	access string // user access token
}

// GetUser fetches the user by ID (login name).
func (c *Client) GetUser(login string) (*remote.User, error) {
	return nil, nil
}

// GetRepos fetches all repositories that the specified
// user has access to in the remote system.
func (c *Client) GetRepos(owner string) ([]*remote.Repo, error) {
	return nil, nil
}

// GetScript fetches the build script (.drone.yml) from the remote
// repository and returns in string format.
func (c *Client) GetScript(*remote.Hook) (string, error) {
	return "", nil
}

// SetStatus
func (c *Client) SetStatus(owner, repo, sha, status string) error {
	return nil
}

// SetActive
func (c *Client) SetActive(owner, repo, hook, key string) error {
	return nil
}
