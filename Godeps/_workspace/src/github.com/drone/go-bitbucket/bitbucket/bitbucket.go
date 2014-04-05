package bitbucket

import (
	"errors"
)

var (
	ErrNilClient = errors.New("client is nil")
)

// New creates an instance of the Bitbucket Client
func New(consumerKey, consumerSecret, accessToken, tokenSecret string) *Client {
	c := &Client{}
	c.ConsumerKey = consumerKey
	c.ConsumerSecret = consumerSecret
	c.AccessToken = accessToken
	c.TokenSecret = tokenSecret

	c.Keys = &KeyResource{c}
	c.Repos = &RepoResource{c}
	c.Users = &UserResource{c}
	c.Emails = &EmailResource{c}
	c.Brokers = &BrokerResource{c}
	c.Teams = &TeamResource{c}
	c.RepoKeys = &RepoKeyResource{c}
	c.Sources = &SourceResource{c}
	return c
}

type Client struct {
	ConsumerKey    string
	ConsumerSecret string
	AccessToken    string
	TokenSecret    string

	Repos    *RepoResource
	Users    *UserResource
	Emails   *EmailResource
	Keys     *KeyResource
	Brokers  *BrokerResource
	Teams    *TeamResource
	Sources  *SourceResource
	RepoKeys *RepoKeyResource
}

// Guest Client that can be used to access
// public APIs that do not require authentication.
var Guest = New("", "", "", "")
