package github

import (
	"fmt"
)

type Key struct {
	Id    int    `json:"id"`
	Key   string `json:"key"`
	Url   string `json:"url"`
	Title string `json:"title"`
}

type KeyResource struct {
	client *Client
}

// Gets a list of the keys associated with an account.
func (r *KeyResource) List() ([]*Key, error) {
	keys := []*Key{}
	const path = "/user/keys"

	if err := r.client.do("GET", path, nil, &keys); err != nil {
		return nil, err
	}

	return keys, nil
}

// Gets the key associated with the specified id.
func (r *KeyResource) Find(id int) (*Key, error) {
	key := Key{}
	path := fmt.Sprintf("/user/keys/%v", id)
	if err := r.client.do("GET", path, nil, &key); err != nil {
		return nil, err
	}

	return &key, nil
}

// Gets the key associated with specified title.
func (r *KeyResource) FindName(title string) (*Key, error) {
	keys, err := r.List()
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

// Creates a key on the specified account. You must supply a valid key
// that is unique across the Github service.
func (r *KeyResource) Create(key, title string) (*Key, error) {

	in := Key{ Title: title, Key: key }
	out := Key{ }
	const path = "/user/keys"
	if err := r.client.do("POST", path, &in, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// Updates a key on the specified account.
func (r *KeyResource) Update(key, title string, id int) (*Key, error) {
	out := Key {}
	in := Key {
		Title : title,
		Key   : key,
	}
	path := fmt.Sprintf("/user/keys/%v", id)
	if err := r.client.do("PATCH", path, &in, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// Creates a key on the specified account, assuming it does not already
// exist in the system
func (r *KeyResource) CreateUpdate(key, title string) (*Key, error) {
	if found, err := r.FindName(title); err == nil {
		// if the public keys are different we should update
		if found.Key != key {
			return r.Update(key, title, found.Id)
		}

		// otherwise we should just return the key, since there
		// is nothing to update
		return found, nil
	}

	return r.Create(key, title)
}

// Deletes the key specified by the id value.
func (r *KeyResource) Delete(id int) error {
	path := fmt.Sprintf("/user/keys/%v", id)
	return r.client.do("DELETE", path, nil, nil)
}

// Deletes the named key.
func (r *KeyResource) DeleteName(title string) error {
	key, err := r.FindName(title)
	if err != nil {
		return err
	}
	return r.Delete(key.Id)
}
