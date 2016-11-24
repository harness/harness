package cache

import (
	"fmt"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
	"golang.org/x/net/context"
)

// GetPerms returns the user permissions repositories from the cache
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

// GetTeamPerms returns the user permissions from the cache
// associated with the current organization.
func GetTeamPerms(c context.Context, user *model.User, org string) (*model.Perm, error) {
	key := fmt.Sprintf("perms:%s:%s",
		user.Login,
		org,
	)
	// if we fetch from the cache we can return immediately
	val, err := Get(c, key)
	if err == nil {
		return val.(*model.Perm), nil
	}
	// else we try to grab from the remote system and
	// populate our cache.
	perm, err := remote.TeamPerm(c, user, org)
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

// GetRepoMap returns the list of user repositories from the cache
// associated with the current context in a map structure.
func GetRepoMap(c context.Context, user *model.User) (map[string]bool, error) {
	repos, err := GetRepos(c, user)
	if err != nil {
		return nil, err
	}
	repom := map[string]bool{}
	for _, repo := range repos {
		repom[repo.FullName] = true
	}
	return repom, nil
}

// DeleteRepos evicts the cached user repositories from the cache associated
// with the current context.
func DeleteRepos(c context.Context, user *model.User) error {
	key := fmt.Sprintf("repos:%s",
		user.Login,
	)
	return Delete(c, key)
}
