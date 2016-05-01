package gitlab

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote/gitlab/testdata"
	"github.com/franela/goblin"
)

func Test_Gitlab(t *testing.T) {
	// setup a dummy github server
	var server = testdata.NewServer()
	defer server.Close()

	env := server.URL + "?client_id=test&client_secret=test"

	gitlab := Load(env)

	var user = model.User{
		Login: "test_user",
		Token: "e3b0c44298fc1c149afbf4c8996fb",
	}

	var repo = model.Repo{
		Name:  "diaspora-client",
		Owner: "diaspora",
	}

	g := goblin.Goblin(t)
	g.Describe("Gitlab Plugin", func() {
		// Test projects method
		g.Describe("AllProjects", func() {
			g.It("Should return only non-archived projects is hidden", func() {
				gitlab.HideArchives = true
				_projects, err := gitlab.Repos(&user)

				g.Assert(err == nil).IsTrue()
				g.Assert(len(_projects)).Equal(1)
			})

			g.It("Should return all the projects", func() {
				gitlab.HideArchives = false
				_projects, err := gitlab.Repos(&user)

				g.Assert(err == nil).IsTrue()
				g.Assert(len(_projects)).Equal(2)
			})
		})

		// Test repository method
		g.Describe("Repo", func() {
			g.It("Should return valid repo", func() {
				_repo, err := gitlab.Repo(&user, "diaspora", "diaspora-client")

				g.Assert(err == nil).IsTrue()
				g.Assert(_repo.Name).Equal("diaspora-client")
				g.Assert(_repo.Owner).Equal("diaspora")
				g.Assert(_repo.IsPrivate).Equal(true)
			})

			g.It("Should return error, when repo not exist", func() {
				_, err := gitlab.Repo(&user, "not-existed", "not-existed")

				g.Assert(err != nil).IsTrue()
			})
		})

		// Test permissions method
		g.Describe("Perm", func() {
			g.It("Should return repo permissions", func() {
				perm, err := gitlab.Perm(&user, "diaspora", "diaspora-client")
				g.Assert(err == nil).IsTrue()
				g.Assert(perm.Admin).Equal(true)
				g.Assert(perm.Pull).Equal(true)
				g.Assert(perm.Push).Equal(true)
			})
			g.It("Should return repo permissions when user is admin", func() {
				perm, err := gitlab.Perm(&user, "brightbox", "puppet")
				g.Assert(err == nil).IsTrue()
				g.Assert(perm.Admin).Equal(true)
				g.Assert(perm.Pull).Equal(true)
				g.Assert(perm.Push).Equal(true)
			})
			g.It("Should return error, when repo is not exist", func() {
				_, err := gitlab.Perm(&user, "not-existed", "not-existed")

				g.Assert(err != nil).IsTrue()
			})
		})

		// Test activate method
		g.Describe("Activate", func() {
			g.It("Should be success", func() {
				err := gitlab.Activate(&user, &repo, "http://example.com/api/hook/test/test?access_token=token")

				g.Assert(err == nil).IsTrue()
			})

			g.It("Should be failed, when token not given", func() {
				err := gitlab.Activate(&user, &repo, "http://example.com/api/hook/test/test")

				g.Assert(err != nil).IsTrue()
			})
		})

		// Test deactivate method
		g.Describe("Deactivate", func() {
			g.It("Should be success", func() {
				err := gitlab.Deactivate(&user, &repo, "http://example.com/api/hook/test/test?access_token=token")

				g.Assert(err == nil).IsTrue()
			})
		})

		// Test login method
		// g.Describe("Login", func() {
		// 	g.It("Should return user", func() {
		// 		user, err := gitlab.Login("valid_token", "")

		// 		g.Assert(err == nil).IsTrue()
		// 		g.Assert(user == nil).IsFalse()
		// 	})

		// 	g.It("Should return error, when token is invalid", func() {
		// 		_, err := gitlab.Login("invalid_token", "")

		// 		g.Assert(err != nil).IsTrue()
		// 	})
		// })

		// Test hook method
		g.Describe("Hook", func() {
			g.Describe("Push hook", func() {
				g.It("Should parse actual push hoook", func() {
					req, _ := http.NewRequest(
						"POST",
						"http://example.com/api/hook?owner=diaspora&name=diaspora-client",
						bytes.NewReader(testdata.PushHook),
					)

					repo, build, err := gitlab.Hook(req)

					g.Assert(err == nil).IsTrue()
					g.Assert(repo.Owner).Equal("mike")
					g.Assert(repo.Name).Equal("diaspora")
					g.Assert(repo.Avatar).Equal("http://example.com/uploads/project/avatar/555/Outh-20-Logo.jpg")
					g.Assert(repo.Branch).Equal("develop")
					g.Assert(build.Ref).Equal("refs/heads/master")

				})

				g.It("Should parse legacy push hoook", func() {
					req, _ := http.NewRequest(
						"POST",
						"http://example.com/api/hook?owner=diaspora&name=diaspora-client",
						bytes.NewReader(testdata.LegacyPushHook),
					)

					repo, build, err := gitlab.Hook(req)

					g.Assert(err == nil).IsTrue()
					g.Assert(repo.Owner).Equal("diaspora")
					g.Assert(repo.Name).Equal("diaspora-client")
					g.Assert(repo.Avatar).Equal("")
					g.Assert(repo.Branch).Equal("master")
					g.Assert(build.Ref).Equal("refs/heads/master")

				})
			})

			g.Describe("Tag push hook", func() {
				g.It("Should parse tag push hook", func() {
					req, _ := http.NewRequest(
						"POST",
						"http://example.com/api/hook?owner=diaspora&name=diaspora-client",
						bytes.NewReader(testdata.TagHook),
					)

					repo, build, err := gitlab.Hook(req)

					g.Assert(err == nil).IsTrue()
					g.Assert(repo.Owner).Equal("jsmith")
					g.Assert(repo.Name).Equal("example")
					g.Assert(repo.Avatar).Equal("http://example.com/uploads/project/avatar/555/Outh-20-Logo.jpg")
					g.Assert(repo.Branch).Equal("develop")
					g.Assert(build.Ref).Equal("refs/tags/v1.0.0")

				})

				g.It("Should parse legacy tag push hook", func() {
					req, _ := http.NewRequest(
						"POST",
						"http://example.com/api/hook?owner=diaspora&name=diaspora-client",
						bytes.NewReader(testdata.LegacyTagHook),
					)

					repo, build, err := gitlab.Hook(req)

					g.Assert(err == nil).IsTrue()
					g.Assert(repo.Owner).Equal("diaspora")
					g.Assert(repo.Name).Equal("diaspora-client")
					g.Assert(build.Ref).Equal("refs/tags/v1.0.0")

				})
			})

			g.Describe("Merge request hook", func() {
				g.It("Should parse merge request hook", func() {
					req, _ := http.NewRequest(
						"POST",
						"http://example.com/api/hook?owner=diaspora&name=diaspora-client",
						bytes.NewReader(testdata.MergeRequestHook),
					)

					repo, build, err := gitlab.Hook(req)

					g.Assert(err == nil).IsTrue()
					g.Assert(repo.Avatar).Equal("http://example.com/uploads/project/avatar/555/Outh-20-Logo.jpg")
					g.Assert(repo.Branch).Equal("develop")
					g.Assert(repo.Owner).Equal("awesome_space")
					g.Assert(repo.Name).Equal("awesome_project")

					g.Assert(build.Title).Equal("MS-Viewport")
				})

				g.It("Should parse legacy merge request hook", func() {
					req, _ := http.NewRequest(
						"POST",
						"http://example.com/api/hook?owner=diaspora&name=diaspora-client",
						bytes.NewReader(testdata.LegacyMergeRequestHook),
					)

					repo, build, err := gitlab.Hook(req)

					g.Assert(err == nil).IsTrue()
					g.Assert(repo.Owner).Equal("diaspora")
					g.Assert(repo.Name).Equal("diaspora-client")

					g.Assert(build.Title).Equal("MS-Viewport")
				})
			})
		})
	})
}
