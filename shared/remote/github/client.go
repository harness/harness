package github

import (
	"fmt"
	"net/url"

	"github.com/drone/drone/shared/remote"
	"github.com/drone/go-github/github"
)

type Client struct {
	config *Github
	access string // user access token
}

// GetUser fetches the user by ID (login name).
func (c *Client) GetUser(login string) (*remote.User, error) {
	return nil, nil
}

// GetRepos fetches all repositories that the specified
// user has access to in the remote system.
func (c *Client) GetRepos(owner string) ([]*remote.Repo, error) {
	// create the github client
	client := github.New(c.access)

	// retrieve a list of all github repositories
	repos, err := client.Repos.ListAll()
	if err != nil {
		return nil, err
	}

	// store results in common format
	result := []*remote.Repo{}

	// parse the hostname from the github url
	githuburl, err := url.Parse(c.config.URL)
	if err != nil {
		return nil, err
	}

	// loop throught the list and convert to the standard repo format
	for _, repo := range repos {
		result = append(result, &remote.Repo{
			ID:      repo.ID,
			Host:    githuburl.Host,
			Owner:   repo.Owner.Login,
			Name:    repo.Name,
			Kind:    "git",
			Clone:   repo.CloneUrl,
			Git:     repo.GitUrl,
			SSH:     repo.SshUrl,
			Private: repo.Private,
			Push:    repo.Permissions.Push,
			Pull:    repo.Permissions.Pull,
			Admin:   repo.Permissions.Admin,
		})
	}

	return result, nil
}

// GetScript fetches the build script (.drone.yml) from the remote
// repository using the GitHub API and returns the raw file in string format.
func (c *Client) GetScript(hook *remote.Hook) (out string, err error) {
	// create the github client
	client := github.New(c.access)

	// retrieve the .drone.yml file from GitHub
	content, err := client.Contents.FindRef(hook.Owner, hook.Repo, ".drone.yml", hook.Sha)
	if err != nil {
		return
	}

	// decode the content
	raw, err := content.DecodeContent()
	if err != nil {
		return
	}

	return string(raw), nil
}

// SetStatus
func (c *Client) SetStatus(owner, name, sha, status string) error {
	// create the github client
	client := github.New(c.access)

	// convert from drone status to github status
	var message string
	switch status {
	case "Success":
		status = "success"
		message = "The build succeeded on drone.io"
	case "Failure":
		status = "failure"
		message = "The build failed on drone.io"
	case "Started", "Pending":
		status = "pending"
		message = "The build is pending on drone.io"
	default:
		status = "error"
		message = "The build errored on drone.io"
	}

	// format the build URL
	// TODO we really need the branch here
	// TODO we really need the drone.io hostname as well
	url := fmt.Sprintf("http://beta.drone.io/%s/%s/%s/%s", owner, name, "master", sha)

	// update the status
	return client.Repos.CreateStatus(owner, name, status, url, message, sha)
}

// SetActive will configure a post-commit and pull-request hook
// with the remote GitHub repository using the GitHub API.
//
// It will also, optionally, add a public RSA key. This is primarily
// used for private repositories, which typically must use the Git+SSH
// protocol to clone the repository.
func (c *Client) SetActive(owner, name, hook, key string) error {
	// create the github client
	client := github.New(c.access)

	// parse the hostname from the hook, and use this
	// to name the ssh key
	hookurl, err := url.Parse(hook)
	if err != nil {
		return err
	}

	// fetch the repository so that we can see if it
	// is public or private.
	repo, err := client.Repos.Find(owner, name)
	if err != nil {
		return err
	}

	// if the repository is private we'll need
	// to upload a github key to the repository
	if repo.Private {
		// name the key
		keyname := "drone@" + hookurl.Host
		_, err := client.RepoKeys.CreateUpdate(owner, name, key, keyname)
		if err != nil {
			return err
		}
	}

	// add the hook
	if _, err := client.Hooks.CreateUpdate(owner, name, hook); err != nil {
		return err
	}

	return nil
}
