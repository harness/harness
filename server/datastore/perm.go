package datastore

import (
	"code.google.com/p/go.net/context"
	"github.com/drone/drone/shared/model"
)

type Permstore interface {
	// GetPerm retrieves the User's permission from
	// the datastore for the given repository.
	GetPerm(user *model.User, repo *model.Repo) (*model.Perm, error)

	// PostPerm saves permission in the datastore.
	PostPerm(perm *model.Perm) error

	// PutPerm saves permission in the datastore.
	PutPerm(perm *model.Perm) error

	// DelPerm removes permission from the datastore.
	DelPerm(perm *model.Perm) error
}

// GetPerm retrieves the User's permission from
// the datastore for the given repository.
func GetPerm(c context.Context, user *model.User, repo *model.Repo) (*model.Perm, error) {
	return FromContext(c).GetPerm(user, repo)
}

// PostPerm saves permission in the datastore.
func PostPerm(c context.Context, perm *model.Perm) error {
	return FromContext(c).PostPerm(perm)
}

// PutPerm saves permission in the datastore.
func PutPerm(c context.Context, perm *model.Perm) error {
	return FromContext(c).PutPerm(perm)
}

// DelPerm removes permission from the datastore.
func DelPerm(c context.Context, perm *model.Perm) error {
	return FromContext(c).DelPerm(perm)
}
