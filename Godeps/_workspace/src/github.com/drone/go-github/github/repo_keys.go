package github

import (
	"fmt"
)

type RepoKeyResource struct {
	client *Client
}

// Gets a list of the keys associated with a repo.
func (r *RepoKeyResource) List(owner, repo string) ([]*Key, error) {
	keys := []*Key{}
	path := fmt.Sprintf("/repos/%s/%s/keys", owner, repo)

	if err := r.client.do("GET", path, nil, &keys); err != nil {
		return nil, err
	}

	return keys, nil
}

// Gets the key associated with the specified id.
func (r *RepoKeyResource) Find(owner, repo string, id int) (*Key, error) {
	key := Key{}
	path := fmt.Sprintf("/repos/%s/%s/keys/%v", owner, repo, id)
	if err := r.client.do("GET", path, nil, &key); err != nil {
		return nil, err
	}

	return &key, nil
}

// Gets the key associated with specified title.
func (r *RepoKeyResource) FindName(owner, repo, title string) (*Key, error) {
	keys, err := r.List(owner, repo)
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		if key.Title == title {
			return key, nil
		}
	}

	return nil, ErrNotFound
}

// Creates a key on the specified repo. You must supply a valid key
// that is unique across the Github service.
func (r *RepoKeyResource) Create(owner, repo, key, title string) (*Key, error) {

	in := Key{Title: title, Key: key}
	out := Key{}
	path := fmt.Sprintf("/repos/%s/%s/keys", owner, repo)
	if err := r.client.do("POST", path, &in, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// Updates a key on the specified repo.
func (r *RepoKeyResource) Update(owner, repo, key, title string, id int) (*Key, error) {
	out := Key{}
	in := Key{
		Title: title,
		Key:   key,
	}
	path := fmt.Sprintf("/repos/%s/%s/keys/%v", owner, repo, id)
	if err := r.client.do("PATCH", path, &in, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// Creates a key for the specified repo, assuming it does not already
// exist in the system
func (r *RepoKeyResource) CreateUpdate(owner, repo, key, title string) (*Key, error) {
	if found, err := r.FindName(owner, repo, title); err == nil {
		// if the public keys are different we should update
		if found.Key != key {
			return r.Update(owner, repo, key, title, found.Id)
		}

		// otherwise we should just return the key, since there
		// is nothing to update
		return found, nil
	}

	return r.Create(owner, repo, key, title)
}

func (r *RepoKeyResource) DeleteName(owner, repo, label string) error {
	key, err := r.FindName(owner, repo, label)
	if err != nil {
		return nil
	}

	return r.Delete(owner, repo, key.Id)
}

// Deletes the key specified by the id value.
func (r *RepoKeyResource) Delete(owner, repo string, id int) error {
	path := fmt.Sprintf("/repos/%s/%s/keys/%v", owner, repo, id)
	return r.client.do("DELETE", path, nil, nil)
}
