package datastore

import (
	"code.google.com/p/go.net/context"
	"github.com/drone/drone/shared/model"
)

type Userstore interface {
	// GetUser retrieves a specific user from the
	// datastore for the given ID.
	GetUser(id int64) (*model.User, error)

	// GetUserLogin retrieves a user from the datastore
	// for the specified remote and login name.
	GetUserLogin(remote, login string) (*model.User, error)

	// GetUserToken retrieves a user from the datastore
	// with the specified token.
	GetUserToken(token string) (*model.User, error)

	// GetUserList retrieves a list of all users from
	// the datastore that are registered in the system.
	GetUserList() ([]*model.User, error)

	// PostUser saves a User in the datastore.
	PostUser(user *model.User) error

	// PutUser saves a user in the datastore.
	PutUser(user *model.User) error

	// DelUser removes the user from the datastore.
	DelUser(user *model.User) error
}

// GetUser retrieves a specific user from the
// datastore for the given ID.
func GetUser(c context.Context, id int64) (*model.User, error) {
	return FromContext(c).GetUser(id)
}

// GetUserLogin retrieves a user from the datastore
// for the specified remote and login name.
func GetUserLogin(c context.Context, remote, login string) (*model.User, error) {
	return FromContext(c).GetUserLogin(remote, login)
}

// GetUserToken retrieves a user from the datastore
// with the specified token.
func GetUserToken(c context.Context, token string) (*model.User, error) {
	return FromContext(c).GetUserToken(token)
}

// GetUserList retrieves a list of all users from
// the datastore that are registered in the system.
func GetUserList(c context.Context) ([]*model.User, error) {
	return FromContext(c).GetUserList()
}

// PostUser saves a User in the datastore.
func PostUser(c context.Context, user *model.User) error {
	return FromContext(c).PostUser(user)
}

// PutUser saves a user in the datastore.
func PutUser(c context.Context, user *model.User) error {
	return FromContext(c).PutUser(user)
}

// DelUser removes the user from the datastore.
func DelUser(c context.Context, user *model.User) error {
	return FromContext(c).DelUser(user)
}
