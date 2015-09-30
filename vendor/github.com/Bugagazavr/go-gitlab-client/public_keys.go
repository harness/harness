package gogitlab

import (
	"encoding/json"
	"net/url"
)

const (
	// ID
	user_keys        = "/user/keys"     // Get current user keys
	user_key         = "/user/keys/:id" // Get user key by id
	custom_user_keys = "/user/:id/keys" // Create key for user with :id
)

type PublicKey struct {
	Id           int    `json:"id,omitempty"`
	Title        string `json:"title,omitempty"`
	Key          string `json:"key,omitempty"`
	CreatedAtRaw string `json:"created_at,omitempty"`
}

func (g *Gitlab) UserKeys() ([]*PublicKey, error) {
	url := g.ResourceUrl(user_keys, nil)
	var keys []*PublicKey
	contents, err := g.buildAndExecRequest("GET", url, nil)
	if err == nil {
		err = json.Unmarshal(contents, &keys)
	}
	return keys, err
}

func (g *Gitlab) UserKey(id string) (*PublicKey, error) {
	url := g.ResourceUrl(user_key, map[string]string{":id": id})
	var key *PublicKey
	contents, err := g.buildAndExecRequest("GET", url, nil)
	if err == nil {
		err = json.Unmarshal(contents, &key)
	}
	return key, err
}

func (g *Gitlab) AddKey(title, key string) error {
	path := g.ResourceUrl(user_keys, nil)
	var err error
	v := url.Values{}
	v.Set("title", title)
	v.Set("key", key)
	body := v.Encode()
	_, err = g.buildAndExecRequest("POST", path, []byte(body))
	return err
}

func (g *Gitlab) AddUserKey(id, title, key string) error {
	path := g.ResourceUrl(user_keys, map[string]string{":id": id})
	var err error
	v := url.Values{}
	v.Set("title", title)
	v.Set("key", key)
	body := v.Encode()
	_, err = g.buildAndExecRequest("POST", path, []byte(body))
	return err
}

func (g *Gitlab) DeleteKey(id string) error {
	url := g.ResourceUrl(user_key, map[string]string{":id": id})
	var err error
	_, err = g.buildAndExecRequest("DELETE", url, nil)
	return err
}
