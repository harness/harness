package store

import (
	"github.com/drone/drone/model"
	"golang.org/x/net/context"
)

type UserStore interface {
	// Get gets a user by unique ID.
	Get(int64) (*model.User, error)

	// GetLogin gets a user by unique Login name.
	GetLogin(string) (*model.User, error)

	// GetList gets a list of all users in the system.
	GetList() ([]*model.User, error)

	// GetFeed gets a user activity feed.
	GetFeed([]*model.RepoLite) ([]*model.Feed, error)

	// Count gets a count of all users in the system.
	Count() (int, error)

	// Create creates a new user account.
	Create(*model.User) error

	// Update updates a user account.
	Update(*model.User) error

	// Delete deletes a user account.
	Delete(*model.User) error
}

func GetUser(c context.Context, id int64) (*model.User, error) {
	return FromContext(c).Users().Get(id)
}

func GetUserLogin(c context.Context, login string) (*model.User, error) {
	return FromContext(c).Users().GetLogin(login)
}

func GetUserList(c context.Context) ([]*model.User, error) {
	return FromContext(c).Users().GetList()
}

func GetUserFeed(c context.Context, listof []*model.RepoLite) ([]*model.Feed, error) {
	return FromContext(c).Users().GetFeed(listof)
}

func CountUsers(c context.Context) (int, error) {
	return FromContext(c).Users().Count()
}

func CreateUser(c context.Context, user *model.User) error {
	return FromContext(c).Users().Create(user)
}

func UpdateUser(c context.Context, user *model.User) error {
	return FromContext(c).Users().Update(user)
}

func DeleteUser(c context.Context, user *model.User) error {
	return FromContext(c).Users().Delete(user)
}
