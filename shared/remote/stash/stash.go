package stash

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/drone/drone/shared/remote"
)

type Stash struct {
	URL     string `json:"url"` // https://bitbucket.org
	API     string `json:"api"` // https://api.bitbucket.org
	Client  string `json:"client"`
	Secret  string `json:"secret"`
	Enabled bool   `json:"enabled"`
}

// GetName returns the name of this remote system.
func (s *Stash) GetName() string {
	return "stash.atlassian.com"
}

// GetHost returns the url.Host of this remote system.
func (s *Stash) GetHost() (host string) {
	u, err := url.Parse(s.URL)
	if err != nil {
		return
	}
	return u.Host
}

// GetHook parses the post-commit hook from the Request body
// and returns the required data in a standard format.
func (s *Stash) GetHook(*http.Request) (*remote.Hook, error) {
	return nil, nil
}

// GetLogin handles authentication to third party, remote services
// and returns the required user data in a standard format.
func (s *Stash) GetLogin(http.ResponseWriter, *http.Request) (*remote.Login, error) {
	return nil, nil
}

// GetClient returns a new Stash remote client.
func (s *Stash) GetClient(access, secret string) remote.Client {
	return &Client{s, access, secret}
}

// IsMatch returns true if the hostname matches the
// hostname of this remote client.
func (s *Stash) IsMatch(hostname string) bool {
	return strings.HasSuffix(hostname, s.URL)
}
