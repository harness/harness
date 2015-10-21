package store

import (
	"github.com/drone/drone/model"
	"golang.org/x/net/context"
)

type KeyStore interface {
	// Get gets a key by unique repository ID.
	Get(*model.Repo) (*model.Key, error)

	// Create creates a new key.
	Create(*model.Key) error

	// Update updates a user key.
	Update(*model.Key) error

	// Delete deletes a user key.
	Delete(*model.Key) error
}

func GetKey(c context.Context, repo *model.Repo) (*model.Key, error) {
	return FromContext(c).Keys().Get(repo)
}

func CreateKey(c context.Context, key *model.Key) error {
	return FromContext(c).Keys().Create(key)
}

func UpdateKey(c context.Context, key *model.Key) error {
	return FromContext(c).Keys().Update(key)
}

func DeleteKey(c context.Context, key *model.Key) error {
	return FromContext(c).Keys().Delete(key)
}
