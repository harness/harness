package cache

import (
	"fmt"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
	"golang.org/x/net/context"
)

// GetPerm returns the user permissions repositories from the cache
// associated with the current repository.
func GetPerms(c context.Context, user *model.User, owner, name string) (*model.Perm, error) {
	key := fmt.Sprintf("perms:%s:%s/%s",
		user.Login,
		owner,
		name,
	)
	// if we fetch from the cache we can return immediately
	val, err := Get(c, key)
	if err == nil {
		return val.(*model.Perm), nil
	}
	// else we try to grab from the remote system and
	// populate our cache.
	perm, err := remote.Perm(c, user, owner, name)
	if err != nil {
		return nil, err
	}
	Set(c, key, perm)
	return perm, nil
}

// GetRepos returns the list of user repositories from the cache
// associated with the current context.
func GetRepos(c context.Context, user *model.User) ([]*model.RepoLite, error) {
	key := fmt.Sprintf("repos:%s",
		user.Login,
	)
	// if we fetch from the cache we can return immediately
	val, err := Get(c, key)
	if err == nil {
		return val.([]*model.RepoLite), nil
	}
	// else we try to grab from the remote system and
	// populate our cache.
	repos, err := remote.Repos(c, user)
	if err != nil {
		return nil, err
	}

	Set(c, key, repos)
	return repos, nil
}
