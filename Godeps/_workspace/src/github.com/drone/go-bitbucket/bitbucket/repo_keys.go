package bitbucket

import (
	"fmt"
	"net/url"
)


// Use the ssh-keys resource to manipulate the ssh-keys associated
// with a repository
// 
// https://confluence.atlassian.com/display/BITBUCKET/deploy-keys+Resource
type RepoKeyResource struct {
	client *Client
}

// Gets a list of the keys associated with a repository.
func (r *RepoKeyResource) List(owner, slug string) ([]*Key, error) {
	keys := []*Key{}
	path := fmt.Sprintf("/repositories/%s/%s/deploy-keys", owner, slug)

	if err := r.client.do("GET", path, nil, nil, &keys); err != nil {
		return nil, err
	}

	return keys, nil
}

// Gets the content of the specified key_id.
// This call requires authentication.
func (r *RepoKeyResource) Find(owner, slug string, id int) (*Key, error) {
	key := Key{}
	path := fmt.Sprintf("/repositories/%s/%s/deploy-keys/%v", owner, slug, id)
	if err := r.client.do("GET", path, nil, nil, &key); err != nil {
		return nil, err
	}

	return &key, nil
}

// Gets the content of the specified key with the
// given label.
func (r *RepoKeyResource) FindName(owner, slug, label string) (*Key, error) {
	keys, err := r.List(owner, slug)
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

// Creates a key on the specified repo. You must supply a valid key
// that is unique across the Bitbucket service.
func (r *RepoKeyResource) Create(owner, slug, key, label string) (*Key, error) {

	values := url.Values{}
	values.Add("key", key)
	values.Add("label", label)

	k := Key{}
	path := fmt.Sprintf("/repositories/%s/%s/deploy-keys", owner, slug)
	if err := r.client.do("POST", path, nil, values, &k); err != nil {
		return nil, err
	}

	return &k, nil
}

// Creates a key on the specified account. You must supply a valid key
// that is unique across the Bitbucket service.
func (r *RepoKeyResource) Update(owner, slug, key, label string, id int) (*Key, error) {

	values := url.Values{}
	values.Add("key", key)
	values.Add("label", label)

	k := Key{}
	path := fmt.Sprintf("/repositories/%s/%s/deploy-keys/%v", owner, slug, id)
	if err := r.client.do("PUT", path, nil, values, &k); err != nil {
		return nil, err
	}

	return &k, nil
}

func (r *RepoKeyResource) CreateUpdate(owner, slug, key, label string) (*Key, error) {
	if found, err := r.FindName(owner, slug, label); err == nil {
		// if the public keys are different we should update
		if found.Key != key {
			return r.Update(owner, slug, key, label, found.Id)
		}

		// otherwise we should just return the key, since there
		// is nothing to update
		return found, nil
	}

	return r.Create(owner, slug, key, label)
}

// Deletes the key specified by the key_id value.
// This call requires authentication
func (r *RepoKeyResource) Delete(owner, slug string, id int) error {
	path := fmt.Sprintf("/repositories/%s/%s/deploy-keys/%v", owner, slug, id)
	return r.client.do("DELETE", path, nil, nil, nil)
}

// Deletes the named key.
// This call requires authentication
func (r *RepoKeyResource) DeleteName(owner, slug, label string) error {
	key, err := r.FindName(owner, slug, label)
	if err != nil {
		return err
	}

	return r.Delete(owner, slug, key.Id)
}
