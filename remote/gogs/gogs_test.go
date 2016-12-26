package gogs

import (
	"net/http/httptest"
	"testing"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote/gogs/fixtures"

	"github.com/franela/goblin"
	"github.com/gin-gonic/gin"
)

func Test_gogs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	s := httptest.NewServer(fixtures.Handler())
	c, _ := New(Opts{
		URL:        s.URL,
		SkipVerify: true,
	})

	g := goblin.Goblin(t)
	g.Describe("Gogs", func() {

		g.After(func() {
			s.Close()
		})

		g.Describe("Creating a remote", func() {
			g.It("Should return client with specified options", func() {
				remote, _ := New(Opts{
					URL:         "http://localhost:8080",
					Username:    "someuser",
					Password:    "password",
					SkipVerify:  true,
					PrivateMode: true,
				})
				g.Assert(remote.(*client).URL).Equal("http://localhost:8080")
				g.Assert(remote.(*client).Machine).Equal("localhost")
				g.Assert(remote.(*client).Username).Equal("someuser")
				g.Assert(remote.(*client).Password).Equal("password")
				g.Assert(remote.(*client).SkipVerify).Equal(true)
				g.Assert(remote.(*client).PrivateMode).Equal(true)
			})
			g.It("Should handle malformed url", func() {
				_, err := New(Opts{URL: "%gh&%ij"})
				g.Assert(err != nil).IsTrue()
			})
		})

		g.Describe("Generating a netrc file", func() {
			g.It("Should return a netrc with the user token", func() {
				remote, _ := New(Opts{
					URL: "http://gogs.com",
				})
				netrc, _ := remote.Netrc(fakeUser, nil)
				g.Assert(netrc.Machine).Equal("gogs.com")
				g.Assert(netrc.Login).Equal(fakeUser.Token)
				g.Assert(netrc.Password).Equal("x-oauth-basic")
			})
			g.It("Should return a netrc with the machine account", func() {
				remote, _ := New(Opts{
					URL:      "http://gogs.com",
					Username: "someuser",
					Password: "password",
				})
				netrc, _ := remote.Netrc(nil, nil)
				g.Assert(netrc.Machine).Equal("gogs.com")
				g.Assert(netrc.Login).Equal("someuser")
				g.Assert(netrc.Password).Equal("password")
			})
		})

		g.Describe("Requesting a repository", func() {
			g.It("Should return the repository details", func() {
				repo, err := c.Repo(fakeUser, fakeRepo.Owner, fakeRepo.Name)
				g.Assert(err == nil).IsTrue()
				g.Assert(repo.Owner).Equal(fakeRepo.Owner)
				g.Assert(repo.Name).Equal(fakeRepo.Name)
				g.Assert(repo.FullName).Equal(fakeRepo.Owner + "/" + fakeRepo.Name)
				g.Assert(repo.IsPrivate).IsTrue()
				g.Assert(repo.Clone).Equal("http://localhost/test_name/repo_name.git")
				g.Assert(repo.Link).Equal("http://localhost/test_name/repo_name")
			})
			g.It("Should handle a not found error", func() {
				_, err := c.Repo(fakeUser, fakeRepoNotFound.Owner, fakeRepoNotFound.Name)
				g.Assert(err != nil).IsTrue()
			})
		})

		g.Describe("Requesting repository permissions", func() {
			g.It("Should return the permission details", func() {
				perm, err := c.Perm(fakeUser, fakeRepo.Owner, fakeRepo.Name)
				g.Assert(err == nil).IsTrue()
				g.Assert(perm.Admin).IsTrue()
				g.Assert(perm.Push).IsTrue()
				g.Assert(perm.Pull).IsTrue()
			})
			g.It("Should handle a not found error", func() {
				_, err := c.Perm(fakeUser, fakeRepoNotFound.Owner, fakeRepoNotFound.Name)
				g.Assert(err != nil).IsTrue()
			})
		})

		g.Describe("Requesting a repository list", func() {
			g.It("Should return the repository list", func() {
				repos, err := c.Repos(fakeUser)
				g.Assert(err == nil).IsTrue()
				g.Assert(repos[0].Owner).Equal(fakeRepo.Owner)
				g.Assert(repos[0].Name).Equal(fakeRepo.Name)
				g.Assert(repos[0].FullName).Equal(fakeRepo.Owner + "/" + fakeRepo.Name)
			})
			g.It("Should handle a not found error", func() {
				_, err := c.Repos(fakeUserNoRepos)
				g.Assert(err != nil).IsTrue()
			})
		})

		g.It("Should register repositroy hooks", func() {
			err := c.Activate(fakeUser, fakeRepo, "http://localhost")
			g.Assert(err == nil).IsTrue()
		})

		g.It("Should return a repository file", func() {
			raw, err := c.File(fakeUser, fakeRepo, fakeBuild, ".drone.yml")
			g.Assert(err == nil).IsTrue()
			g.Assert(string(raw)).Equal("{ platform: linux/amd64 }")
		})

		g.It("Should return a repository file from a ref", func() {
			raw, err := c.File(fakeUser, fakeRepo, fakeBuildWithRef, ".drone.yml")
			g.Assert(err == nil).IsTrue()
			g.Assert(string(raw)).Equal("{ platform: linux/amd64 }")
		})

		g.Describe("Given an authentication request", func() {
			g.It("Should redirect to login form")
			g.It("Should create an access token")
			g.It("Should handle an access token error")
			g.It("Should return the authenticated user")
		})

		g.Describe("Given a repository hook", func() {
			g.It("Should skip non-push events")
			g.It("Should return push details")
			g.It("Should handle a parsing error")
		})

		g.It("Should return no-op for usupporeted features", func() {
			_, err1 := c.Auth("octocat", "4vyW6b49Z")
			err2 := c.Status(nil, nil, nil, "")
			err3 := c.Deactivate(nil, nil, "")
			g.Assert(err1 != nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
		})
	})
}

var (
	fakeUser = &model.User{
		Login: "someuser",
		Token: "cfcd2084",
	}

	fakeUserNoRepos = &model.User{
		Login: "someuser",
		Token: "repos_not_found",
	}

	fakeRepo = &model.Repo{
		Owner:    "test_name",
		Name:     "repo_name",
		FullName: "test_name/repo_name",
	}

	fakeRepoNotFound = &model.Repo{
		Owner:    "test_name",
		Name:     "repo_not_found",
		FullName: "test_name/repo_not_found",
	}

	fakeBuild = &model.Build{
		Commit: "9ecad50",
	}

	fakeBuildWithRef = &model.Build{
		Ref: "refs/tags/v1.0.0",
	}
)
