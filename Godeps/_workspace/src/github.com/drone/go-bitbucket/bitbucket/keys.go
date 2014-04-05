package bitbucket

import (
	"fmt"
	"net/url"
)

type Key struct {
	Id    int    `json:"pk"`    // The key identifier (ID).
	Key   string `json:"key"`   // Public key value.
	Label string `json:"label"` // The user-visible label on the key
}

// Use the ssh-keys resource to manipulate the ssh-keys on an individual
// or team account.
// 
// https://confluence.atlassian.com/display/BITBUCKET/ssh-keys+Resource
type KeyResource struct {
	client *Client
}

// Gets a list of the keys associated with an account.
// This call requires authentication.
func (r *KeyResource) List(account string) ([]*Key, error) {
	keys := []*Key{}
	path := fmt.Sprintf("/users/%s/ssh-keys", account)

	if err := r.client.do("GET", path, nil, nil, &keys); err != nil {
		return nil, err
	}

	return keys, nil
}

// Gets the content of the specified key_id.
// This call requires authentication.
func (r *KeyResource) Find(account string, id int) (*Key, error) {
	key := Key{}
	path := fmt.Sprintf("/users/%s/ssh-keys/%v", account, id)
	if err := r.client.do("GET", path, nil, nil, &key); err != nil {
		return nil, err
	}

	return &key, nil
}

// Gets the content of the specified key with the
// given label.
func (r *KeyResource) FindName(account, label string) (*Key, error) {
	keys, err := r.List(account)
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		if key.Label == label {
			return key, nil
		}
	}

	return nil, ErrNotFound
}

// Creates a key on the specified account. You must supply a valid key
// that is unique across the Bitbucket service.
func (r *KeyResource) Create(account, key, label string) (*Key, error) {

	values := url.Values{}
	values.Add("key", key)
	values.Add("label", label)

	k := Key{}
	path := fmt.Sprintf("/users/%s/ssh-keys", account)
	if err := r.client.do("POST", path, nil, values, &k); err != nil {
		return nil, err
	}

	return &k, nil
}

// Creates a key on the specified account. You must supply a valid key
// that is unique across the Bitbucket service.
func (r *KeyResource) Update(account, key, label string, id int) (*Key, error) {

	values := url.Values{}
	values.Add("key", key)
	values.Add("label", label)

	k := Key{}
	path := fmt.Sprintf("/users/%s/ssh-keys/%v", account, id)
	if err := r.client.do("PUT", path, nil, values, &k); err != nil {
		return nil, err
	}

	return &k, nil
}

func (r *KeyResource) CreateUpdate(account, key, label string) (*Key, error) {
	if found, err := r.FindName(account, label); err == nil {
		// if the public keys are different we should update
		if found.Key != key {
			return r.Update(account, key, label, found.Id)
		}

		// otherwise we should just return the key, since there
		// is nothing to update
		return found, nil
	}

	return r.Create(account, key, label)
}

// Deletes the key specified by the key_id value.
// This call requires authentication
func (r *KeyResource) Delete(account string, id int) error {
	path := fmt.Sprintf("/users/%s/ssh-keys/%v", account, id)
	return r.client.do("DELETE", path, nil, nil, nil)
}
