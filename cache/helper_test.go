package cache

import (
	"errors"
	"fmt"
	"testing"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/remote/mock"
	"github.com/franela/goblin"
	"github.com/gin-gonic/gin"
)

func TestHelper(t *testing.T) {

	g := goblin.Goblin(t)

	g.Describe("Cache helpers", func() {

		var c *gin.Context
		var r *mock.Remote

		g.BeforeEach(func() {
			c = new(gin.Context)
			ToContext(c, Default())

			r = new(mock.Remote)
			remote.ToContext(c, r)
		})

		g.It("Should get permissions from remote", func() {
			r.On("Perm", fakeUser, fakeRepo.Owner, fakeRepo.Name).Return(fakePerm, nil).Once()
			p, err := GetPerms(c, fakeUser, fakeRepo.Owner, fakeRepo.Name)
			g.Assert(p).Equal(fakePerm)
			g.Assert(err).Equal(nil)

		})

		g.It("Should get permissions from cache", func() {
			key := fmt.Sprintf("perms:%s:%s/%s",
				fakeUser.Login,
				fakeRepo.Owner,
				fakeRepo.Name,
			)

			Set(c, key, fakePerm)
			r.On("Perm", fakeUser, fakeRepo.Owner, fakeRepo.Name).Return(nil, fakeErr).Once()
			p, err := GetPerms(c, fakeUser, fakeRepo.Owner, fakeRepo.Name)
			g.Assert(p).Equal(fakePerm)
			g.Assert(err).Equal(nil)
		})

		g.It("Should get permissions error", func() {
			r.On("Perm", fakeUser, fakeRepo.Owner, fakeRepo.Name).Return(nil, fakeErr).Once()
			p, err := GetPerms(c, fakeUser, fakeRepo.Owner, fakeRepo.Name)
			g.Assert(p == nil).IsTrue()
			g.Assert(err).Equal(fakeErr)
		})

		g.It("Should set and get repos", func() {

			r.On("Repos", fakeUser).Return(fakeRepos, nil).Once()
			p, err := GetRepos(c, fakeUser)
			g.Assert(p).Equal(fakeRepos)
			g.Assert(err).Equal(nil)
		})

		g.It("Should get repos", func() {
			key := fmt.Sprintf("repos:%s",
				fakeUser.Login,
			)

			Set(c, key, fakeRepos)
			r.On("Repos", fakeUser).Return(nil, fakeErr).Once()
			p, err := GetRepos(c, fakeUser)
			g.Assert(p).Equal(fakeRepos)
			g.Assert(err).Equal(nil)
		})

		g.It("Should get repos error", func() {
			r.On("Repos", fakeUser).Return(nil, fakeErr).Once()
			p, err := GetRepos(c, fakeUser)
			g.Assert(p == nil).IsTrue()
			g.Assert(err).Equal(fakeErr)
		})

		g.It("Should evict repos", func() {
			key := fmt.Sprintf("repos:%s",
				fakeUser.Login,
			)

			Set(c, key, fakeRepos)
			repos, err := Get(c, key)
			g.Assert(repos != nil).IsTrue()
			g.Assert(err == nil).IsTrue()

			DeleteRepos(c, fakeUser)
			repos, err = Get(c, key)
			g.Assert(repos == nil).IsTrue()
		})
	})
}

var (
	fakeErr   = errors.New("Not Found")
	fakeUser  = &model.User{Login: "octocat"}
	fakePerm  = &model.Perm{true, true, true}
	fakeRepo  = &model.RepoLite{Owner: "octocat", Name: "Hello-World"}
	fakeRepos = []*model.RepoLite{
		{Owner: "octocat", Name: "Hello-World"},
		{Owner: "octocat", Name: "hello-world"},
		{Owner: "octocat", Name: "Spoon-Knife"},
	}
)
