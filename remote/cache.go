package remote

import (
	"time"

	"github.com/drone/drone/model"
)

// WithCache returns a the parent Remote with a front-end Cache. Remote items
// are cached for duration d.
func WithCache(r Remote, d time.Duration) Remote {
	return r
}

// Cacher implements purge functionality so that we can evict stale data and
// force a refresh. The indended use case is when the repository list is out
// of date and requires manual refresh.
type Cacher interface {
	Purge(*model.User)
}

// Because the cache is so closely tied to the remote we should just include
// them in the same package together. The below code are stubs for merging
// the Cache with the Remote package.

type cache struct {
	Remote
}

func (c *cache) Repos(u *model.User) ([]*model.RepoLite, error) {
	// key := fmt.Sprintf("repos:%s",
	// 	user.Login,
	// )
	// // if we fetch from the cache we can return immediately
	// val, err := Get(c, key)
	// if err == nil {
	// 	return val.([]*model.RepoLite), nil
	// }
	// // else we try to grab from the remote system and
	// // populate our cache.
	// repos, err := remote.Repos(c, user)
	// if err != nil {
	// 	return nil, err
	// }
	//
	// Set(c, key, repos)
	// return repos, nil
	return nil, nil
}

func (c *cache) Perm(u *model.User, owner, repo string) (*model.Perm, error) {
	// key := fmt.Sprintf("perms:%s:%s/%s",
	// 	user.Login,
	// 	owner,
	// 	name,
	// )
	// // if we fetch from the cache we can return immediately
	// val, err := Get(c, key)
	// if err == nil {
	// 	return val.(*model.Perm), nil
	// }
	// // else we try to grab from the remote system and
	// // populate our cache.
	// perm, err := remote.Perm(c, user, owner, name)
	// if err != nil {
	// 	return nil, err
	// }
	// Set(c, key, perm)
	// return perm, nil
	return nil, nil
}

func (c *cache) Purge(*model.User) {
	return
}

func (c *cache) Refresh(u *model.User) (bool, error) {
	if r, ok := c.Remote.(Refresher); ok {
		return r.Refresh(u)
	}
	return false, nil
}

var _ Remote = &cache{}
var _ Refresher = &cache{}
