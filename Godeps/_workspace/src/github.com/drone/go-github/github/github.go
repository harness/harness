package github

// New creates an instance of the Github Client
func New(token string) *Client {
	c := &Client{}
	c.Token = token

	c.Keys = &KeyResource{c}
	c.Repos = &RepoResource{c}
	c.Users = &UserResource{c}
	c.Orgs = &OrgResource{c}
	c.Emails = &EmailResource{c}
	c.Hooks = &HookResource{c}
	c.Contents = &ContentResource{c}
	c.RepoKeys = &RepoKeyResource{c}
	c.ApiUrl = "https://api.github.com"
	return c
}

type Client struct {
	ApiUrl string
	Token string

	Repos    *RepoResource
	Users    *UserResource
	Orgs     *OrgResource
	Emails   *EmailResource
	Keys     *KeyResource
	Hooks    *HookResource
	Contents *ContentResource
	RepoKeys *RepoKeyResource
}

// Guest Client that can be used to access
// public APIs that do not require authentication.
var Guest = New("")
