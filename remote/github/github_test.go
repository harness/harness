package github

import (
	"net/http/httptest"
	"testing"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote/github/fixtures"

	"github.com/franela/goblin"
	"github.com/gin-gonic/gin"
)

func Test_github(t *testing.T) {
	gin.SetMode(gin.TestMode)

	s := httptest.NewServer(fixtures.Handler())
	c, _ := New(Opts{
		URL:        s.URL,
		SkipVerify: true,
	})

	g := goblin.Goblin(t)
	g.Describe("GitHub", func() {

		g.After(func() {
			s.Close()
		})

		g.Describe("Creating a remote", func() {
			g.It("Should return client with specified options", func() {
				remote, _ := New(Opts{
					URL:         "http://localhost:8080/",
					Client:      "0ZXh0IjoiI",
					Secret:      "I1NiIsInR5",
					Username:    "someuser",
					Password:    "password",
					SkipVerify:  true,
					PrivateMode: true,
					Context:     "continuous-integration/test",
				})
				g.Assert(remote.(*client).URL).Equal("http://localhost:8080")
				g.Assert(remote.(*client).API).Equal("http://localhost:8080/api/v3/")
				g.Assert(remote.(*client).Machine).Equal("localhost")
				g.Assert(remote.(*client).Username).Equal("someuser")
				g.Assert(remote.(*client).Password).Equal("password")
				g.Assert(remote.(*client).Client).Equal("0ZXh0IjoiI")
				g.Assert(remote.(*client).Secret).Equal("I1NiIsInR5")
				g.Assert(remote.(*client).SkipVerify).Equal(true)
				g.Assert(remote.(*client).PrivateMode).Equal(true)
				g.Assert(remote.(*client).Context).Equal("continuous-integration/test")
			})
			g.It("Should handle malformed url", func() {
				_, err := New(Opts{URL: "%gh&%ij"})
				g.Assert(err != nil).IsTrue()
			})
		})

		g.Describe("Generating a netrc file", func() {
			g.It("Should return a netrc with the user token", func() {
				remote, _ := New(Opts{
					URL: "http://github.com:443",
				})
				netrc, _ := remote.Netrc(fakeUser, nil)
				g.Assert(netrc.Machine).Equal("github.com")
				g.Assert(netrc.Login).Equal(fakeUser.Token)
				g.Assert(netrc.Password).Equal("x-oauth-basic")
			})
			g.It("Should return a netrc with the machine account", func() {
				remote, _ := New(Opts{
					URL:      "http://github.com:443",
					Username: "someuser",
					Password: "password",
				})
				netrc, _ := remote.Netrc(nil, nil)
				g.Assert(netrc.Machine).Equal("github.com")
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
				g.Assert(repo.FullName).Equal(fakeRepo.FullName)
				g.Assert(repo.IsPrivate).IsTrue()
				g.Assert(repo.Clone).Equal(fakeRepo.Clone)
				g.Assert(repo.Link).Equal(fakeRepo.Link)
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

		g.Describe("Requesting organization permissions", func() {
			g.It("Should return the permission details of an admin", func() {
				perm, err := c.TeamPerm(fakeUser, "octocat")
				g.Assert(err == nil).IsTrue()
				g.Assert(perm.Admin).IsTrue()
			})
			g.It("Should return the permission details of a member", func() {
				perm, err := c.TeamPerm(fakeUser, "github")
				g.Assert(err == nil).IsTrue()
				g.Assert(perm.Admin).IsFalse()
			})
			g.It("Should handle a not found error", func() {
				_, err := c.TeamPerm(fakeUser, "org_not_found")
				g.Assert(err != nil).IsTrue()
			})
		})

		g.It("Should return a user repository list")

		g.It("Should return a user team list")

		g.It("Should register repositroy hooks")

		g.It("Should return a repository file")

		g.Describe("Given an authentication request", func() {
			g.It("Should redirect to the GitHub login page")
			g.It("Should create an access token")
			g.It("Should handle an access token error")
			g.It("Should return the authenticated user")
			g.It("Should handle authentication errors")
		})
	})
}

var (
	fakeUser = &model.User{
		Login: "octocat",
		Token: "cfcd2084",
	}

	fakeUserNoRepos = &model.User{
		Login: "octocat",
		Token: "repos_not_found",
	}

	fakeRepo = &model.Repo{
		Owner:     "octocat",
		Name:      "Hello-World",
		FullName:  "octocat/Hello-World",
		Avatar:    "https://github.com/images/error/octocat_happy.gif",
		Link:      "https://github.com/octocat/Hello-World",
		Clone:     "https://github.com/octocat/Hello-World.git",
		IsPrivate: true,
	}

	fakeRepoNotFound = &model.Repo{
		Owner:    "test_name",
		Name:     "repo_not_found",
		FullName: "test_name/repo_not_found",
	}

	fakeBuild = &model.Build{
		Commit: "9ecad50",
	}
)
