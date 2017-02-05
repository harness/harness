package bitbucket

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote/bitbucket/fixtures"
	"github.com/drone/drone/remote/bitbucket/internal"

	"github.com/franela/goblin"
	"github.com/gin-gonic/gin"
)

func Test_bitbucket(t *testing.T) {
	gin.SetMode(gin.TestMode)

	s := httptest.NewServer(fixtures.Handler())
	c := &config{URL: s.URL, API: s.URL}

	g := goblin.Goblin(t)
	g.Describe("Bitbucket client", func() {

		g.After(func() {
			s.Close()
		})

		g.It("Should return client with default endpoint", func() {
			remote := New("4vyW6b49Z", "a5012f6c6")
			g.Assert(remote.(*config).URL).Equal(DefaultURL)
			g.Assert(remote.(*config).API).Equal(DefaultAPI)
			g.Assert(remote.(*config).Client).Equal("4vyW6b49Z")
			g.Assert(remote.(*config).Secret).Equal("a5012f6c6")
		})

		g.It("Should return the netrc file", func() {
			remote := New("", "")
			netrc, _ := remote.Netrc(fakeUser, nil)
			g.Assert(netrc.Machine).Equal("bitbucket.org")
			g.Assert(netrc.Login).Equal("x-token-auth")
			g.Assert(netrc.Password).Equal(fakeUser.Token)
		})

		g.Describe("Given an authorization request", func() {
			g.It("Should redirect to authorize", func() {
				w := httptest.NewRecorder()
				r, _ := http.NewRequest("GET", "", nil)
				_, err := c.Login(w, r)
				g.Assert(err == nil).IsTrue()
				g.Assert(w.Code).Equal(http.StatusSeeOther)
			})
			g.It("Should return authenticated user", func() {
				r, _ := http.NewRequest("GET", "?code=code", nil)
				u, err := c.Login(nil, r)
				g.Assert(err == nil).IsTrue()
				g.Assert(u.Login).Equal(fakeUser.Login)
				g.Assert(u.Token).Equal("2YotnFZFEjr1zCsicMWpAA")
				g.Assert(u.Secret).Equal("tGzv3JOkF0XG5Qx2TlKWIA")
			})
			g.It("Should handle failure to exchange code", func() {
				w := httptest.NewRecorder()
				r, _ := http.NewRequest("GET", "?code=code_bad_request", nil)
				_, err := c.Login(w, r)
				g.Assert(err != nil).IsTrue()
			})
			g.It("Should handle failure to resolve user", func() {
				r, _ := http.NewRequest("GET", "?code=code_user_not_found", nil)
				_, err := c.Login(nil, r)
				g.Assert(err != nil).IsTrue()
			})
			g.It("Should handle authentication errors", func() {
				r, _ := http.NewRequest("GET", "?error=invalid_scope", nil)
				_, err := c.Login(nil, r)
				g.Assert(err != nil).IsTrue()
			})
		})

		g.Describe("Given an access token", func() {
			g.It("Should return the authenticated user", func() {
				login, err := c.Auth(
					fakeUser.Token,
					fakeUser.Secret,
				)
				g.Assert(err == nil).IsTrue()
				g.Assert(login).Equal(fakeUser.Login)
			})
			g.It("Should handle a failure to resolve user", func() {
				_, err := c.Auth(
					fakeUserNotFound.Token,
					fakeUserNotFound.Secret,
				)
				g.Assert(err != nil).IsTrue()
			})
		})

		g.Describe("Given a refresh token", func() {
			g.It("Should return a refresh access token", func() {
				ok, err := c.Refresh(fakeUserRefresh)
				g.Assert(err == nil).IsTrue()
				g.Assert(ok).IsTrue()
				g.Assert(fakeUserRefresh.Token).Equal("2YotnFZFEjr1zCsicMWpAA")
				g.Assert(fakeUserRefresh.Secret).Equal("tGzv3JOkF0XG5Qx2TlKWIA")
			})
			g.It("Should handle an empty access token", func() {
				ok, err := c.Refresh(fakeUserRefreshEmpty)
				g.Assert(err == nil).IsTrue()
				g.Assert(ok).IsFalse()
			})
			g.It("Should handle a failure to refresh", func() {
				ok, err := c.Refresh(fakeUserRefreshFail)
				g.Assert(err != nil).IsTrue()
				g.Assert(ok).IsFalse()
			})
		})

		g.Describe("When requesting a repository", func() {
			g.It("Should return the details", func() {
				repo, err := c.Repo(
					fakeUser,
					fakeRepo.Owner,
					fakeRepo.Name,
				)
				g.Assert(err == nil).IsTrue()
				g.Assert(repo.FullName).Equal(fakeRepo.FullName)
			})
			g.It("Should handle not found errors", func() {
				_, err := c.Repo(
					fakeUser,
					fakeRepoNotFound.Owner,
					fakeRepoNotFound.Name,
				)
				g.Assert(err != nil).IsTrue()
			})
		})

		g.Describe("When requesting repository permissions", func() {
			g.It("Should handle not found errors", func() {
				_, err := c.Perm(
					fakeUser,
					fakeRepoNotFound.Owner,
					fakeRepoNotFound.Name,
				)
				g.Assert(err != nil).IsTrue()
			})
			g.It("Should authorize read access", func() {
				perm, err := c.Perm(
					fakeUser,
					fakeRepoNoHooks.Owner,
					fakeRepoNoHooks.Name,
				)
				g.Assert(err == nil).IsTrue()
				g.Assert(perm.Pull).IsTrue()
				g.Assert(perm.Push).IsFalse()
				g.Assert(perm.Admin).IsFalse()
			})
			g.It("Should authorize admin access", func() {
				perm, err := c.Perm(
					fakeUser,
					fakeRepo.Owner,
					fakeRepo.Name,
				)
				g.Assert(err == nil).IsTrue()
				g.Assert(perm.Pull).IsTrue()
				g.Assert(perm.Push).IsTrue()
				g.Assert(perm.Admin).IsTrue()
			})
		})

		g.Describe("When requesting user repositories", func() {
			g.It("Should return the details", func() {
				repos, err := c.Repos(fakeUser)
				g.Assert(err == nil).IsTrue()
				g.Assert(repos[0].FullName).Equal(fakeRepo.FullName)
			})
			g.It("Should handle organization not found errors", func() {
				_, err := c.Repos(fakeUserNoTeams)
				g.Assert(err != nil).IsTrue()
			})
			g.It("Should handle not found errors", func() {
				_, err := c.Repos(fakeUserNoRepos)
				g.Assert(err != nil).IsTrue()
			})
		})

		g.Describe("When requesting user teams", func() {
			g.It("Should return the details", func() {
				teams, err := c.Teams(fakeUser)
				g.Assert(err == nil).IsTrue()
				g.Assert(teams[0].Login).Equal("superfriends")
				g.Assert(teams[0].Avatar).Equal("http://i.imgur.com/ZygP55A.jpg")
			})
			g.It("Should handle not found error", func() {
				_, err := c.Teams(fakeUserNoTeams)
				g.Assert(err != nil).IsTrue()
			})
		})

		g.Describe("When downloading a file", func() {
			g.It("Should return the bytes", func() {
				raw, err := c.File(fakeUser, fakeRepo, fakeBuild, "file")
				g.Assert(err == nil).IsTrue()
				g.Assert(len(raw) != 0).IsTrue()
			})
			g.It("Should handle not found error", func() {
				_, err := c.File(fakeUser, fakeRepo, fakeBuild, "file_not_found")
				g.Assert(err != nil).IsTrue()
			})
		})

		g.Describe("When activating a repository", func() {
			g.It("Should error when malformed hook", func() {
				err := c.Activate(fakeUser, fakeRepo, "%gh&%ij")
				g.Assert(err != nil).IsTrue()
			})
			g.It("Should create the hook", func() {
				err := c.Activate(fakeUser, fakeRepo, "http://127.0.0.1")
				g.Assert(err == nil).IsTrue()
			})
		})

		g.Describe("When deactivating a repository", func() {
			g.It("Should error when listing hooks fails", func() {
				err := c.Deactivate(fakeUser, fakeRepoNoHooks, "http://127.0.0.1")
				g.Assert(err != nil).IsTrue()
			})
			g.It("Should successfully remove hooks", func() {
				err := c.Deactivate(fakeUser, fakeRepo, "http://127.0.0.1")
				g.Assert(err == nil).IsTrue()
			})
			g.It("Should successfully deactivate when hook already removed", func() {
				err := c.Deactivate(fakeUser, fakeRepoEmptyHook, "http://127.0.0.1")
				g.Assert(err == nil).IsTrue()
			})
		})

		g.Describe("Given a list of hooks", func() {
			g.It("Should return the matching hook", func() {
				hooks := []*internal.Hook{
					{Url: "http://127.0.0.1/hook"},
				}
				hook := matchingHooks(hooks, "http://127.0.0.1/")
				g.Assert(hook).Equal(hooks[0])
			})
			g.It("Should handle no matches", func() {
				hooks := []*internal.Hook{
					{Url: "http://localhost/hook"},
				}
				hook := matchingHooks(hooks, "http://127.0.0.1/")
				g.Assert(hook == nil).IsTrue()
			})
			g.It("Should handle malformed hook urls", func() {
				var hooks []*internal.Hook
				hook := matchingHooks(hooks, "%gh&%ij")
				g.Assert(hook == nil).IsTrue()
			})
		})

		g.It("Should update the status", func() {
			err := c.Status(fakeUser, fakeRepo, fakeBuild, "http://127.0.0.1")
			g.Assert(err == nil).IsTrue()
		})

		g.It("Should parse the hook", func() {
			buf := bytes.NewBufferString(fixtures.HookPush)
			req, _ := http.NewRequest("POST", "/hook", buf)
			req.Header = http.Header{}
			req.Header.Set(hookEvent, hookPush)

			r, _, err := c.Hook(req)
			g.Assert(err == nil).IsTrue()
			g.Assert(r.FullName).Equal("user_name/repo_name")
		})

	})
}

var (
	fakeUser = &model.User{
		Login: "superman",
		Token: "cfcd2084",
	}

	fakeUserRefresh = &model.User{
		Login:  "superman",
		Secret: "cfcd2084",
	}

	fakeUserRefreshFail = &model.User{
		Login:  "superman",
		Secret: "refresh_token_not_found",
	}

	fakeUserRefreshEmpty = &model.User{
		Login:  "superman",
		Secret: "refresh_token_is_empty",
	}

	fakeUserNotFound = &model.User{
		Login: "superman",
		Token: "user_not_found",
	}

	fakeUserNoTeams = &model.User{
		Login: "superman",
		Token: "teams_not_found",
	}

	fakeUserNoRepos = &model.User{
		Login: "superman",
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

	fakeRepoNoHooks = &model.Repo{
		Owner:    "test_name",
		Name:     "hooks_not_found",
		FullName: "test_name/hooks_not_found",
	}

	fakeRepoEmptyHook = &model.Repo{
		Owner:    "test_name",
		Name:     "hook_empty",
		FullName: "test_name/hook_empty",
	}

	fakeBuild = &model.Build{
		Commit: "9ecad50",
	}
)
