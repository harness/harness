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
	// if the user is a guest they should only be granted
	// read access to public repositories.
	switch {
	case user == nil && repo.Private:
		return &model.Perm{
			Guest: true,
			Read:  false,
			Write: false,
			Admin: false}, nil
	case user == nil && !repo.Private:
		return &model.Perm{
			Guest: true,
			Read:  true,
			Write: false,
			Admin: false}, nil
	}

	// if the user is authenticated we'll retireive the
	// permission details from the database.
	perm, err := FromContext(c).GetPerm(user, repo)
	if perm.ID == 0 {
		perm.Guest = true
	}

	switch {
	// if the user is a system admin grant super access.
	case user.Admin == true:
		perm.Read = true
		perm.Write = true
		perm.Admin = true

	// if the repo is public, grant read access only.
	case repo.Private == false:
		perm.Read = true
	}
	return perm, err
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
