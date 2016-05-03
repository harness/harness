package github

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/google/go-github/github"
)

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
	if !strings.HasPrefix(*user.URL, defaultAPI) && len(emails) != 0 {
		user.Email = emails[0].Email
		return user, nil
	}

	return nil, fmt.Errorf("No verified Email address for GitHub account")
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
		hookurl, ok := hook.Config["url"].(string)
		if !ok {
			continue
		}
		if strings.HasPrefix(hookurl, url) {
			return &hook, nil
		}
	}
	return nil, nil
}

// DeleteHook does exactly what you think it does.
func DeleteHook(client *github.Client, owner, name, url string) error {
	hook, err := GetHook(client, owner, name, url)
	if err != nil {
		return err
	}
	if hook == nil {
		return nil
	}
	_, err = client.Repositories.DeleteHook(owner, name, *hook.ID)
	return err
}

// CreateHook is a heper function that creates a post-commit hook
// for the specified repository.
func CreateHook(client *github.Client, owner, name, url string) (*github.Hook, error) {
	var hook = new(github.Hook)
	hook.Name = github.String("web")
	hook.Events = []string{"push", "pull_request", "deployment"}
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

// GetPayload is a helper function that will parse the JSON payload. It will
// first check for a `payload` parameter in a POST, but can fallback to a
// raw JSON body as well.
func GetPayload(req *http.Request) []byte {
	var payload = req.FormValue("payload")
	if len(payload) == 0 {
		defer req.Body.Close()
		raw, _ := ioutil.ReadAll(req.Body)
		return raw
	}
	return []byte(payload)
}
