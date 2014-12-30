package gitlab

import (
	"fmt"
	"net/url"

	"github.com/Bugagazavr/go-gitlab-client"
)

// NewClient is a helper function that returns a new GitHub
// client using the provided OAuth token.
func NewClient(uri, token string, skipVerify bool) *gogitlab.Gitlab {
	return gogitlab.NewGitlabCert(uri, "/api/v3", token, skipVerify)
}

// IsRead is a helper function that returns true if the
// user has Read-only access to the repository.
func IsRead(proj *gogitlab.Project) bool {
	var user = proj.Permissions.ProjectAccess
	var group = proj.Permissions.GroupAccess

	switch {
	case proj.Public:
		return true
	case user != nil && user.AccessLevel >= 20:
		return true
	case group != nil && group.AccessLevel >= 20:
		return true
	default:
		return false
	}
}

// IsWrite is a helper function that returns true if the
// user has Read-Write access to the repository.
func IsWrite(proj *gogitlab.Project) bool {
	var user = proj.Permissions.ProjectAccess
	var group = proj.Permissions.GroupAccess

	switch {
	case user != nil && user.AccessLevel >= 30:
		return true
	case group != nil && group.AccessLevel >= 30:
		return true
	default:
		return false
	}
}

// IsAdmin is a helper function that returns true if the
// user has Admin access to the repository.
func IsAdmin(proj *gogitlab.Project) bool {
	var user = proj.Permissions.ProjectAccess
	var group = proj.Permissions.GroupAccess

	switch {
	case user != nil && user.AccessLevel >= 40:
		return true
	case group != nil && group.AccessLevel >= 40:
		return true
	default:
		return false
	}
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

func ns(owner, name string) string {
	return fmt.Sprintf("%s%%2F%s", owner, name)
}
