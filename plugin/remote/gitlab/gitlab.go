package gitlab

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/Bugagazavr/go-gitlab-client"
	"github.com/drone/drone/plugin/remote"
	"github.com/drone/drone/shared/model"
)

type Gitlab struct {
	URL     string `json:"url"` // https://github.com
	Enabled bool   `json:"enabled"`
}

// GetName returns the name of this remote system.
func (g *Gitlab) GetName() string {
	return "gitlab.com"
}

// GetHost returns the url.Host of this remote system.
func (g *Gitlab) GetHost() (host string) {
	u, err := url.Parse(g.URL)
	if err != nil {
		return
	}
	return u.Host
}

// GetHook parses the post-commit hook from the Request body
// and returns the required data in a standard format.
func (g *Gitlab) GetHook(r *http.Request, u *model.User) (*remote.Hook, error) {
	owner := r.FormValue(":owner")
	name := r.FormValue(":name")

	payload, _ := ioutil.ReadAll(r.Body)
	parsed, err := gogitlab.ParseHook(payload)
	if err != nil {
		return nil, err
	}

	if parsed.ObjectKind == "merge_request" {
		return g.GetPullRequestHook(r, u)
	}

	if len(parsed.After) == 0 {
		return nil, nil
	}

	hook := remote.Hook{}

	hook.Repo = name
	hook.Owner = owner
	hook.Sha = parsed.After
	hook.Branch = parsed.Branch()

	head := parsed.Head()

	hook.Message = head.Message
	hook.Timestamp = head.Timestamp
	if head.Author != nil {
		hook.Author = head.Author.Email
	} else {
		hook.Author = parsed.UserName
	}

	return &hook, nil
}

func (g *Gitlab) GetPullRequestHook(*http.Request, *model.User) (*remote.Hook, error) {
	return nil, nil
}

// GetLogin handles authentication to third party, remote services
// and returns the required user data in a standard format.
func (g *Gitlab) GetLogin(w http.ResponseWriter, r *http.Request) (*remote.Login, error) {
	user_login := r.FormValue("login")
	user_password := r.FormValue("password")

	client := gogitlab.NewGitlab(g.URL, "/api/v3", "")
	session, err := client.GetSession(user_login, user_password)
	if err != nil {
		redirect := "/login"
		http.Redirect(w, r, redirect, http.StatusUnauthorized)
		return nil, err
	}

	login := remote.Login{
		ID:     int64(session.Id),
		Login:  session.UserName,
		Access: session.PrivateToken,
		Name:   session.Name,
		Email:  session.Email,
	}

	return &login, nil
}

// GetClient returns a new Gitlab remote client.
func (g *Gitlab) GetClient(access, secret string) remote.Client {
	return &Client{g, access}
}

// IsMatch returns true if the hostname matches the
// hostname of this remote client.
func (g *Gitlab) IsMatch(hostname string) bool {
	return strings.HasSuffix(hostname, g.URL)
}
