package cache

import (
	"fmt"

	"github.com/drone/drone/model"
	"golang.org/x/net/context"
)

// GetRepos returns the user permissions to the named repository
// from the cache associated with the current context.
func GetPerms(c context.Context, user *model.User, owner, name string) *model.Perm {
	key := fmt.Sprintf("perms:%s:%s/%s",
		user.Login,
		owner,
		name,
	)
	val, err := FromContext(c).Get(key)
	if err != nil {
		return nil
	}
	return val.(*model.Perm)
}

// SetRepos adds the listof user permissions to the named repsotiory
// to the cache assocaited with the current context.
func SetPerms(c context.Context, user *model.User, perm *model.Perm, owner, name string) {
	key := fmt.Sprintf("perms:%s:%s/%s",
		user.Login,
		owner,
		name,
	)
	FromContext(c).Set(key, perm)
}

// GetRepos returns the list of user repositories from the cache
// associated with the current context.
func GetRepos(c context.Context, user *model.User) []*model.RepoLite {
	key := fmt.Sprintf("repos:%s",
		user.Login,
	)
	val, err := FromContext(c).Get(key)
	if err != nil {
		return nil
	}
	return val.([]*model.RepoLite)
}

// SetRepos adds the listof user repositories to the cache assocaited
// with the current context.
func SetRepos(c context.Context, user *model.User, repos []*model.RepoLite) {
	key := fmt.Sprintf("repos:%s",
		user.Login,
	)
	FromContext(c).Set(key, repos)
}

// GetSetRepos is a helper function that attempts to get the
// repository list from the cache first. If no data is in the
// cache or it is expired, it will remotely fetch the list of
// repositories and populate the cache.
// func GetSetRepos(c context.Context, user *model.User) ([]*model.RepoLite, error) {
// 	cache := FromContext(c).Repos()
// 	repos := FromContext(c).Repos().Get(user)
// 	if repos != nil {
// 		return repos, nil
// 	}
// 	var err error
// 	repos, err = remote.FromContext(c).Repos(user)
// 	if err != nil {
// 		return nil, err
// 	}
// 	cache.Set(user, repos)
// 	return repos, nil
// }
