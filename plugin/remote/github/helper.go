package github

import (
	"crypto/tls"
	"encoding/base32"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/drone/drone/plugin/remote/github/oauth"
	"github.com/google/go-github/github"
	"github.com/gorilla/securecookie"
)

// NewClient is a helper function that returns a new GitHub
// client using the provided OAuth token.
func NewClient(uri, token string, skipVerify bool) *github.Client {
	t := &oauth.Transport{
		Token: &oauth.Token{AccessToken: token},
	}

	// this is for GitHub enterprise users that are using
	// self-signed certificates.
	if skipVerify {
		t.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	c := github.NewClient(t.Client())
	c.BaseURL, _ = url.Parse(uri)
	return c
}

// GetUserEmail is a heper function that retrieves the currently
// authenticated user from GitHub + Email address.
func GetUserEmail(client *github.Client) (*github.User, error) {
	user, _, err := client.Users.Get("")
	if err != nil {
		return nil, err
	}

	emails, _, err := client.Users.ListEmails(nil)
	if err != nil {
		return nil, err
	}

	for _, email := range emails {
		if *email.Primary && *email.Verified {
			user.Email = email.Email
			return user, nil
		}
	}

	// WARNING, HACK
	// for out-of-date github enterprise editions the primary
	// and verified fields won't exist.
	if !strings.HasPrefix(*user.HTMLURL, DefaultURL) && len(emails) != 0 {
		user.Email = emails[0].Email
		return user, nil
	}

	return nil, fmt.Errorf("No verified Email address for GitHub account")
}

// GetAllRepos is a helper function that returns an aggregated list
// of all user and organization repositories.
func GetAllRepos(client *github.Client) ([]github.Repository, error) {
	orgs, err := GetOrgs(client)
	if err != nil {
		return nil, err
	}

	repos, err := GetUserRepos(client)
	if err != nil {
		return nil, err
	}

	for _, org := range orgs {
		list, err := GetOrgRepos(client, *org.Login)
		if err != nil {
			return nil, err
		}
		repos = append(repos, list...)
	}

	return repos, nil
}

// GetUserRepos is a helper function that returns a list of
// all user repositories. Paginated results are aggregated into
// a single list.
func GetUserRepos(client *github.Client) ([]github.Repository, error) {
	var repos []github.Repository
	var opts = github.RepositoryListOptions{}
	opts.PerPage = 100
	opts.Page = 1

	// loop through user repository list
	for opts.Page > 0 {
		list, resp, err := client.Repositories.List("", &opts)
		if err != nil {
			return nil, err
		}
		repos = append(repos, list...)

		// increment the next page to retrieve
		opts.Page = resp.NextPage
	}

	return repos, nil
}

// GetOrgRepos is a helper function that returns a list of
// all org repositories. Paginated results are aggregated into
// a single list.
func GetOrgRepos(client *github.Client, org string) ([]github.Repository, error) {
	var repos []github.Repository
	var opts = github.RepositoryListByOrgOptions{}
	opts.PerPage = 100
	opts.Page = 1

	// loop through user repository list
	for opts.Page > 0 {
		list, resp, err := client.Repositories.ListByOrg(org, &opts)
		if err != nil {
			return nil, err
		}
		repos = append(repos, list...)

		// increment the next page to retrieve
		opts.Page = resp.NextPage
	}

	return repos, nil
}

// GetOrgs is a helper function that returns a list of
// all orgs that a user belongs to.
func GetOrgs(client *github.Client) ([]github.Organization, error) {
	var orgs []github.Organization
	var opts = github.ListOptions{}
	opts.Page = 1

	for opts.Page > 0 {
		list, resp, err := client.Organizations.List("", &opts)
		if err != nil {
			return nil, err
		}
		orgs = append(orgs, list...)

		// increment the next page to retrieve
		opts.Page = resp.NextPage
	}
	return orgs, nil
}

// GetHook is a heper function that retrieves a hook by
// hostname. To do this, it will retrieve a list of all hooks
// and iterate through the list.
func GetHook(client *github.Client, owner, name, url string) (*github.Hook, error) {
	hooks, _, err := client.Repositories.ListHooks(owner, name, nil)
	if err != nil {
		return nil, err
	}
	for _, hook := range hooks {
		if hook.Config["url"] == url {
			return &hook, nil
		}
	}
	return nil, nil
}

func DeleteHook(client *github.Client, owner, name, url string) error {
	hook, err := GetHook(client, owner, name, url)
	if err != nil {
		return err
	}

	_, err = client.Repositories.DeleteHook(owner, name, *hook.ID)
	return err
}

// CreateHook is a heper function that creates a post-commit hook
// for the specified repository.
func CreateHook(client *github.Client, owner, name, url string) (*github.Hook, error) {
	var hook = new(github.Hook)
	hook.Name = github.String("web")
	hook.Events = []string{"push", "pull_request"}
	hook.Config = map[string]interface{}{}
	hook.Config["url"] = url
	hook.Config["content_type"] = "form"
	created, _, err := client.Repositories.CreateHook(owner, name, hook)
	return created, err
}

// CreateUpdateHook is a heper function that creates a post-commit hook
// for the specified repository if it does not already exist, otherwise
// it updates the existing hook
func CreateUpdateHook(client *github.Client, owner, name, url string) (*github.Hook, error) {
	var hook, _ = GetHook(client, owner, name, url)
	if hook != nil {
		hook.Name = github.String("web")
		hook.Events = []string{"push", "pull_request"}
		hook.Config = map[string]interface{}{}
		hook.Config["url"] = url
		hook.Config["content_type"] = "form"
		var updated, _, err = client.Repositories.EditHook(owner, name, *hook.ID, hook)
		return updated, err
	}

	return CreateHook(client, owner, name, url)
}

// GetKey is a heper function that retrieves a public Key by
// title. To do this, it will retrieve a list of all keys
// and iterate through the list.
func GetKey(client *github.Client, owner, name, title string) (*github.Key, error) {
	keys, _, err := client.Repositories.ListKeys(owner, name, nil)
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		if *key.Title == title {
			return &key, nil
		}
	}
	return nil, nil
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

// DeleteKey is a helper function that deletes a deploy key
// for the specified repository.
func DeleteKey(client *github.Client, owner, name, title, key string) error {
	var k, err = GetKey(client, owner, name, title)
	if err != nil {
		return err
	}
	_, err = client.Repositories.DeleteKey(owner, name, *k.ID)
	return err
}

// CreateKey is a helper function that creates a deploy key
// for the specified repository.
func CreateKey(client *github.Client, owner, name, title, key string) (*github.Key, error) {
	var k = new(github.Key)
	k.Title = github.String(title)
	k.Key = github.String(key)
	created, _, err := client.Repositories.CreateKey(owner, name, k)
	return created, err
}

// CreateUpdateKey is a helper function that creates a deployment key
// for the specified repository if it does not already exist, otherwise
// it updates the existing key
func CreateUpdateKey(client *github.Client, owner, name, title, key string) (*github.Key, error) {
	var k, _ = GetKey(client, owner, name, title)
	if k != nil {
		k.Title = github.String(title)
		k.Key = github.String(key)
		client.Repositories.DeleteKey(owner, name, *k.ID)
	}

	return CreateKey(client, owner, name, title, key)
}

// GetFile is a heper function that retrieves a file from
// GitHub and returns its contents in byte array format.
func GetFile(client *github.Client, owner, name, path, ref string) ([]byte, error) {
	var opts = new(github.RepositoryContentGetOptions)
	opts.Ref = ref
	content, _, _, err := client.Repositories.GetContents(owner, name, path, opts)
	if err != nil {
		return nil, err
	}
	return content.Decode()
}

// GetRandom is a helper function that generates a 32-bit random
// key, base32 encoded as a string value.
func GetRandom() string {
	return base32.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(32))
}

// GetPayload is a helper function that will parse the JSON payload. It will
// first check for a `payload` parameter in a POST, but can fallback to a
// raw JSON body as well.
func GetPayload(req *http.Request) []byte {
	var payload = req.FormValue("payload")
	if len(payload) == 0 {
		raw, _ := ioutil.ReadAll(req.Body)
		return raw
	}
	return []byte(payload)
}

// UserBelongsToOrg returns true if the currently authenticated user is a
// member of any of the organizations provided.
func UserBelongsToOrg(client *github.Client, permittedOrgs []string) (bool, error) {
	userOrgs, err := GetOrgs(client)
	if err != nil {
		return false, err
	}

	userOrgSet := make(map[string]struct{}, len(userOrgs))
	for _, org := range userOrgs {
		userOrgSet[*org.Login] = struct{}{}
	}

	for _, org := range permittedOrgs {
		if _, ok := userOrgSet[org]; ok {
			return true, nil
		}
	}

	return false, nil
}
