package bitbucket

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/drone/drone/plugin/remote"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/go-bitbucket/bitbucket"
	"github.com/drone/go-bitbucket/oauth1"
)

type Bitbucket struct {
	URL     string `json:"url"` // https://bitbucket.org
	API     string `json:"api"` // https://api.bitbucket.org
	Client  string `json:"client"`
	Secret  string `json:"secret"`
	Enabled bool   `json:"enabled"`
}

// GetName returns the name of this remote system.
func (b *Bitbucket) GetName() string {
	return "bitbucket.org"
}

// GetHost returns the url.Host of this remote system.
func (b *Bitbucket) GetHost() (host string) {
	u, err := url.Parse(b.URL)
	if err != nil {
		return
	}
	return u.Host
}

// GetHook parses the post-commit hook from the Request body
// and returns the required data in a standard format.
func (b *Bitbucket) GetHook(r *http.Request) (*remote.Hook, error) {
	// get the payload from the request
	payload := r.FormValue("payload")

	// parse the post-commit hook
	hook, err := bitbucket.ParseHook([]byte(payload))
	if err != nil {
		return nil, err
	}

	// verify the payload has the minimum amount of required data.
	if hook.Repo == nil || hook.Commits == nil || len(hook.Commits) == 0 {
		return nil, fmt.Errorf("Invalid Bitbucket post-commit Hook. Missing Repo or Commit data.")
	}

	return &remote.Hook{
		Owner:     hook.Repo.Owner,
		Repo:      hook.Repo.Name,
		Sha:       hook.Commits[len(hook.Commits)-1].Hash,
		Branch:    hook.Commits[len(hook.Commits)-1].Branch,
		Author:    hook.Commits[len(hook.Commits)-1].Author,
		Timestamp: time.Now().UTC().String(),
		Message:   hook.Commits[len(hook.Commits)-1].Message,
	}, nil
}

// GetLogin handles authentication to third party, remote services
// and returns the required user data in a standard format.
func (b *Bitbucket) GetLogin(w http.ResponseWriter, r *http.Request) (*remote.Login, error) {

	// bitbucket oauth1 consumer
	consumer := oauth1.Consumer{
		RequestTokenURL:  "https://bitbucket.org/api/1.0/oauth/request_token/",
		AuthorizationURL: "https://bitbucket.org/!api/1.0/oauth/authenticate",
		AccessTokenURL:   "https://bitbucket.org/api/1.0/oauth/access_token/",
		CallbackURL:      httputil.GetScheme(r) + "://" + httputil.GetHost(r) + "/login/bitbucket.org",
		ConsumerKey:      b.Client,
		ConsumerSecret:   b.Secret,
	}

	// get the oauth verifier
	verifier := r.FormValue("oauth_verifier")
	if len(verifier) == 0 {
		// Generate a Request Token
		requestToken, err := consumer.RequestToken()
		if err != nil {
			return nil, err
		}

		// add the request token as a signed cookie
		httputil.SetCookie(w, r, "bitbucket_token", requestToken.Encode())

		url, _ := consumer.AuthorizeRedirect(requestToken)
		http.Redirect(w, r, url, http.StatusSeeOther)
		return nil, nil
	}

	// remove bitbucket token data once before redirecting
	// back to the application.
	defer httputil.DelCookie(w, r, "bitbucket_token")

	// get the tokens from the request
	requestTokenStr := httputil.GetCookie(r, "bitbucket_token")
	requestToken, err := oauth1.ParseRequestTokenStr(requestTokenStr)
	if err != nil {
		return nil, err
	}

	// exchange for an access token
	accessToken, err := consumer.AuthorizeToken(requestToken, verifier)
	if err != nil {
		return nil, err
	}

	// create the Bitbucket client
	client := bitbucket.New(
		b.Client,
		b.Secret,
		accessToken.Token(),
		accessToken.Secret(),
	)

	// get the currently authenticated Bitbucket User
	user, err := client.Users.Current()
	if err != nil {
		return nil, err
	}

	// put the user data in the common format
	login := remote.Login{
		Login:  user.User.Username,
		Access: accessToken.Token(),
		Secret: accessToken.Secret(),
		Name:   user.User.DisplayName,
	}

	email, _ := client.Emails.FindPrimary(user.User.Username)
	if email != nil {
		login.Email = email.Email
	}

	return &login, nil
}

// GetClient returns a new Bitbucket remote client.
func (b *Bitbucket) GetClient(access, secret string) remote.Client {
	return &Client{b, access, secret}
}

// IsMatch returns true if the hostname matches the
// hostname of this remote client.
func (b *Bitbucket) IsMatch(hostname string) bool {
	return hostname == "bitbucket.org"
}
