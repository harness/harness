package phabricator

import (
	"github.com/drone/drone/remote/phabricator/client"
)

func NewClient(url, accessToken string, skipVerify bool) *client.Client {
	return client.New(url, accessToken, skipVerify)
}
