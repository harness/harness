package gitlab

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/franela/goblin"
	"github.com/drone/drone/pkg/remote/builtin/gitlab/testdata"
	"github.com/drone/drone/pkg/types"
)

func Test_Gitlab(t *testing.T) {
	// setup a dummy github server
	var server = testdata.NewServer()
	defer server.Close()

	var gitlab, err = NewDriver(server.URL + "?client_id=test&client_secret=test")
	if err != nil {
		panic(err)
	}

	var user = types.User{
		Login: "test_user",
		Token: "e3b0c44298fc1c149afbf4c8996fb",
	}

	var repo = types.Repo{
		Name:  "diaspora-client",
		Owner: "diaspora",
	}

	g := goblin.Goblin(t)
	g.Describe("Gitlab Plugin", func() {
		// Test repository method
		g.Describe("Repo", func() {
			g.It("Should return valid repo", func() {
				_repo, err := gitlab.Repo(&user, "diaspora", "diaspora-client")

				g.Assert(err == nil).IsTrue()
				g.Assert(_repo.Name).Equal("diaspora-client")
				g.Assert(_repo.Owner).Equal("diaspora")
				g.Assert(_repo.Private).Equal(true)
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

				fmt.Println(gitlab.(*Gitlab), err)
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
				err := gitlab.Activate(&user, &repo, &types.Keypair{}, "http://example.com/api/hook/test/test?access_token=token")

				g.Assert(err == nil).IsTrue()
			})

			g.It("Should be failed, when token not given", func() {
				err := gitlab.Activate(&user, &repo, &types.Keypair{}, "http://example.com/api/hook/test/test")

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
		g.Describe("Login", func() {
			g.It("Should return user", func() {
				user, err := gitlab.Login("valid_token", "")

				g.Assert(err == nil).IsTrue()
				g.Assert(user == nil).IsFalse()
			})

			g.It("Should return error, when token is invalid", func() {
				_, err := gitlab.Login("invalid_token", "")

				g.Assert(err != nil).IsTrue()
			})
		})

		// Test hook method
		g.Describe("Hook", func() {
			g.It("Should parse push hoook", func() {
				req, _ := http.NewRequest(
					"POST",
					"http://example.com/api/hook?owner=diaspora&name=diaspora-client",
					bytes.NewReader(testdata.PushHook),
				)

				hook, err := gitlab.Hook(req)

				g.Assert(err == nil).IsTrue()
				g.Assert(hook.Repo.Owner).Equal("diaspora")
				g.Assert(hook.Repo.Name).Equal("diaspora-client")
				g.Assert(hook.Commit.Ref).Equal("refs/heads/master")

				g.Assert(hook.PullRequest == nil).IsTrue()
			})

			g.It("Should parse tag push hook", func() {
				req, _ := http.NewRequest(
					"POST",
					"http://example.com/api/hook?owner=diaspora&name=diaspora-client",
					bytes.NewReader(testdata.TagHook),
				)

				hook, err := gitlab.Hook(req)

				g.Assert(err == nil).IsTrue()
				g.Assert(hook.Repo.Owner).Equal("diaspora")
				g.Assert(hook.Repo.Name).Equal("diaspora-client")
				g.Assert(hook.Commit.Ref).Equal("refs/tags/v1.0.0")

				g.Assert(hook.PullRequest == nil).IsTrue()
			})

			g.It("Should parse merge request hook", func() {
				req, _ := http.NewRequest(
					"POST",
					"http://example.com/api/hook?owner=diaspora&name=diaspora-client",
					bytes.NewReader(testdata.MergeRequestHook),
				)

				hook, err := gitlab.Hook(req)

				g.Assert(err == nil).IsTrue()
				g.Assert(hook.Repo.Owner).Equal("diaspora")
				g.Assert(hook.Repo.Name).Equal("diaspora-client")

				g.Assert(hook.PullRequest.Number).Equal(1)
				g.Assert(hook.PullRequest.Title).Equal("MS-Viewport")
			})
		})
	})
}
