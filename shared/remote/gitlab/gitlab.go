package gitlab

import (
	"net/http"
	"strings"

	"github.com/drone/drone/shared/remote"
)

type Gitlab struct {
	URL     string `json:"url"` // https://github.com
	Enabled bool   `json:"enabled"`
}

// GetName returns the name of this remote system.
func (g *Gitlab) GetName() string {
	return "gitlab.com"
}

// GetHook parses the post-commit hook from the Request body
// and returns the required data in a standard format.
func (g *Gitlab) GetHook(*http.Request) (*remote.Hook, error) {
	return nil, nil
}

// GetLogin handles authentication to third party, remote services
// and returns the required user data in a standard format.
func (g *Gitlab) GetLogin(http.ResponseWriter, *http.Request) (*remote.Login, error) {
	return nil, nil
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
