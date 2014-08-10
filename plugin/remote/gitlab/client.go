package gitlab

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/Bugagazavr/go-gitlab-client"
	"github.com/drone/drone/plugin/remote"
)

type Client struct {
	config *Gitlab
	access string // user access token
}

type State struct {
	Private      bool
	CloneUrl     string
	PushAbility  bool
	PullAbility  bool
	AdminAbility bool
	Permissions  int
}

// GetUser fetches the user by ID (login name).
func (c *Client) GetUser(login string) (*remote.User, error) {
	return nil, nil
}

// GetRepos fetches all repositories that the specified
// user has access to in the remote system.
func (c *Client) GetRepos(owner string) ([]*remote.Repo, error) {
	// create a new gitlab client
	client := gogitlab.NewGitlab(c.config.URL, "/api/v3", c.access)

	// retrieve a list of all gitlab repositories
	repos, err := client.AllProjects()
	if err != nil {
		return nil, err
	}

	// store results in common format
	result := []*remote.Repo{}

	// parse the hostname from the github url
	gitlaburl, err := url.Parse(c.config.URL)
	if err != nil {
		return nil, err
	}
	// loop throught the list and convert to the standard repo format
	for _, repo := range repos {
		state := &State{}

		if repo.Public {
			state.Private = false
			state.CloneUrl = repo.HttpRepoUrl
		} else {
			state.Private = true
			state.CloneUrl = repo.SshRepoUrl
		}

		// Fetch current repo for permissions
		current_repo, err := client.Project(strconv.Itoa(repo.Id))
		if err != nil {
			return nil, err
		}

		project_access := current_repo.Permissions.ProjectAccess
		group_access := current_repo.Permissions.GroupAccess

		if project_access != nil {
			state.Permissions = project_access.AccessLevel
		} else {
			// If user group member
			if group_access != nil {
				state.Permissions = group_access.AccessLevel
			} else {
				state.Permissions = 0
			}
		}

		// Check access level to set push ability
		if state.Permissions >= 30 {
			state.PushAbility = true
		} else {
			state.PushAbility = false
		}

		// Check access level and project to set pull ability
		if repo.Public || state.Permissions >= 20 {
			state.PullAbility = true
		} else {
			state.PullAbility = false
		}

		// Check access level to set admin ability
		if state.Permissions >= 40 {
			state.AdminAbility = true
		} else {
			state.AdminAbility = false
		}

		result = append(result, &remote.Repo{
			ID:      int64(repo.Id),
			Host:    gitlaburl.Host,
			Owner:   repo.Namespace.Name,
			Name:    repo.Name,
			Kind:    "git",
			Clone:   state.CloneUrl,
			Git:     repo.HttpRepoUrl,
			SSH:     repo.SshRepoUrl,
			Private: state.Private,
			Push:    state.PushAbility,
			Pull:    state.PullAbility,
			Admin:   state.AdminAbility,
		})
	}

	return result, nil
}

// GetScript fetches the build script (.drone.yml) from the remote
// repository and returns in string format.
func (c *Client) GetScript(hook *remote.Hook) (string, error) {
	client := gogitlab.NewGitlab(c.config.URL, "/api/v3", c.access)

	// create repo path
	path := ns(hook.Owner, hook.Repo)

	content, err := client.RepoRawFile(path, hook.Sha, ".drone.yml")
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// SetStatus
func (c *Client) SetStatus(owner, repo, sha, status string) error {
	return nil
}

// SetActive
func (c *Client) SetActive(owner, name, hook, key string) error {
	// create a new gitlab client
	client := gogitlab.NewGitlab(c.config.URL, "/api/v3", c.access)

	// parse the hostname from the hook, and use this
	// to name the ssh key
	hookurl, err := url.Parse(hook)
	if err != nil {
		return err
	}

	// create repo path
	path := ns(owner, name)

	// fetch the repository so that we can see if it
	// is public or private.
	repo, err := client.Project(path)
	if err != nil {
		return err
	}

	// if the repository is private we'll need
	// to upload a github key to the repository
	if !repo.Public {
		keyname := "drone@" + hookurl.Host
		if err := client.AddProjectDeployKey(path, keyname, key); err != nil {
			return err
		}
	}

	// add the hook
	if err := client.AddProjectHook(path, hook, true, false, true); err != nil {
		return err
	}

	return nil
}

func ns(user, repo string) string {
	return fmt.Sprintf("%s%%2F%s", user, repo)
}
