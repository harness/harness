package bitbucket

import (
	"fmt"
	"github.com/drone/drone/shared/remote"
	"github.com/drone/go-bitbucket/bitbucket"
	"net/url"
)

type Client struct {
	config *Bitbucket
	access string // user access token
	secret string // user access token secret
}

// GetUser fetches the user by ID (login name).
func (c *Client) GetUser(login string) (*remote.User, error) {
	return nil, nil
}

// GetRepos fetches all repositories that the specified
// user has access to in the remote system.
func (c *Client) GetRepos(owner string) ([]*remote.Repo, error) {
	// create the Bitbucket client
	client := bitbucket.New(
		c.config.Client,
		c.config.Secret,
		c.access,
		c.secret,
	)

	// parse the hostname from the bitbucket url
	bitbucketurl, err := url.Parse(c.config.URL)
	if err != nil {
		return nil, err
	}

	repos, err := client.Repos.List()
	if err != nil {
		return nil, err
	}

	// store results in common format
	result := []*remote.Repo{}

	// loop throught the list and convert to the standard repo format
	for _, repo := range repos {
		// for now we only support git repos
		if repo.Scm != "git" {
			continue
		}

		// these are the urls required to clone the repository
		// TODO use the bitbucketurl.Host and bitbucketurl.Scheme instead of hardcoding
		//      so that we can support Stash.
		clone := fmt.Sprintf("https://bitbucket.org/%s/%s.git", repo.Owner, repo.Name)
		ssh := fmt.Sprintf("git@bitbucket.org:%s/%s.git", repo.Owner, repo.Name)

		result = append(result, &remote.Repo{
			Host:    bitbucketurl.Host,
			Owner:   repo.Owner,
			Name:    repo.Name,
			Kind:    repo.Scm,
			Private: repo.Private,
			Clone:   clone,
			SSH:     ssh,
			// Bitbucket doesn't return permissions with repository
			// lists, so we're going to grant full access.
			//
			// TODO we need to verify this API call only returns
			//      repositories that we can access (ie not repos we just follow).
			//      otherwise this would cause a security flaw.
			Push:  true,
			Pull:  true,
			Admin: true,
		})
	}

	return result, nil
}

// GetScript fetches the build script (.drone.yml) from the remote
// repository and returns in string format.
func (c *Client) GetScript(hook *remote.Hook) (out string, err error) {
	// create the Bitbucket client
	client := bitbucket.New(
		c.config.Client,
		c.config.Secret,
		c.access,
		c.secret,
	)

	// get the yaml from the database
	raw, err := client.Sources.Find(hook.Owner, hook.Repo, hook.Sha, ".drone.yml")
	if err != nil {
		return
	}

	return raw.Data, nil
}

// SetStatus
func (c *Client) SetStatus(owner, name, sha, status string) error {
	// not implemented for Bitbucket
	return nil
}

// SetActive
func (c *Client) SetActive(owner, name, hook, key string) error {
	// create the Bitbucket client
	client := bitbucket.New(
		c.config.Client,
		c.config.Secret,
		c.access,
		c.secret,
	)

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
	_, err = client.Brokers.CreateUpdate(owner, name, hook, bitbucket.BrokerTypePost)
	return err
}
