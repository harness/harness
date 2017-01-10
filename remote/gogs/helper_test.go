package gogs

import (
	"bytes"
	"testing"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote/gogs/fixtures"

	"github.com/franela/goblin"
	"github.com/gogits/go-gogs-client"
)

func Test_parse(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Gogs", func() {

		g.It("Should parse push hook payload", func() {
			buf := bytes.NewBufferString(fixtures.HookPush)
			hook, err := parsePush(buf)
			g.Assert(err == nil).IsTrue()
			g.Assert(hook.Ref).Equal("refs/heads/master")
			g.Assert(hook.After).Equal("ef98532add3b2feb7a137426bba1248724367df5")
			g.Assert(hook.Before).Equal("4b2626259b5a97b6b4eab5e6cca66adb986b672b")
			g.Assert(hook.Compare).Equal("http://gogs.golang.org/gordon/hello-world/compare/4b2626259b5a97b6b4eab5e6cca66adb986b672b...ef98532add3b2feb7a137426bba1248724367df5")
			g.Assert(hook.Repo.Name).Equal("hello-world")
			g.Assert(hook.Repo.URL).Equal("http://gogs.golang.org/gordon/hello-world")
			g.Assert(hook.Repo.Owner.Name).Equal("gordon")
			g.Assert(hook.Repo.FullName).Equal("gordon/hello-world")
			g.Assert(hook.Repo.Owner.Email).Equal("gordon@golang.org")
			g.Assert(hook.Repo.Owner.Username).Equal("gordon")
			g.Assert(hook.Repo.Private).Equal(true)
			g.Assert(hook.Pusher.Name).Equal("gordon")
			g.Assert(hook.Pusher.Email).Equal("gordon@golang.org")
			g.Assert(hook.Pusher.Username).Equal("gordon")
			g.Assert(hook.Sender.Login).Equal("gordon")
			g.Assert(hook.Sender.Avatar).Equal("http://gogs.golang.org///1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87")
		})

		g.It("Should parse tag hook payload", func() {
			buf := bytes.NewBufferString(fixtures.HookPushTag)
			hook, err := parsePush(buf)
			g.Assert(err == nil).IsTrue()
			g.Assert(hook.Ref).Equal("v1.0.0")
			g.Assert(hook.Repo.Name).Equal("hello-world")
			g.Assert(hook.Repo.URL).Equal("http://gogs.golang.org/gordon/hello-world")
			g.Assert(hook.Repo.FullName).Equal("gordon/hello-world")
			g.Assert(hook.Repo.Owner.Email).Equal("gordon@golang.org")
			g.Assert(hook.Repo.Owner.Username).Equal("gordon")
			g.Assert(hook.Repo.Private).Equal(true)
			g.Assert(hook.Sender.Username).Equal("gordon")
			g.Assert(hook.Sender.Avatar).Equal("https://secure.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87")
		})

		g.It("Should parse pull_request hook payload", func() {
			buf := bytes.NewBufferString(fixtures.HookPullRequest)
			hook, err := parsePullRequest(buf)
			g.Assert(err == nil).IsTrue()
			g.Assert(hook.Action).Equal("opened")
			g.Assert(hook.Number).Equal(int64(1))

			g.Assert(hook.Repo.Name).Equal("hello-world")
			g.Assert(hook.Repo.URL).Equal("http://gogs.golang.org/gordon/hello-world")
			g.Assert(hook.Repo.FullName).Equal("gordon/hello-world")
			g.Assert(hook.Repo.Owner.Email).Equal("gordon@golang.org")
			g.Assert(hook.Repo.Owner.Username).Equal("gordon")
			g.Assert(hook.Repo.Private).Equal(true)
			g.Assert(hook.Sender.Username).Equal("gordon")
			g.Assert(hook.Sender.Avatar).Equal("https://secure.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87")

			g.Assert(hook.PullRequest.Title).Equal("Update the README with new information")
			g.Assert(hook.PullRequest.Body).Equal("please merge")
			g.Assert(hook.PullRequest.State).Equal("open")
			g.Assert(hook.PullRequest.User.Username).Equal("gordon")
			g.Assert(hook.PullRequest.Base.Label).Equal("master")
			g.Assert(hook.PullRequest.Base.Ref).Equal("master")
			g.Assert(hook.PullRequest.Head.Label).Equal("feature/changes")
			g.Assert(hook.PullRequest.Head.Ref).Equal("feature/changes")
		})

		g.It("Should return a Build struct from a push hook", func() {
			buf := bytes.NewBufferString(fixtures.HookPush)
			hook, _ := parsePush(buf)
			build := buildFromPush(hook)
			g.Assert(build.Event).Equal(model.EventPush)
			g.Assert(build.Commit).Equal(hook.After)
			g.Assert(build.Ref).Equal(hook.Ref)
			g.Assert(build.Link).Equal(hook.Compare)
			g.Assert(build.Branch).Equal("master")
			g.Assert(build.Message).Equal(hook.Commits[0].Message)
			g.Assert(build.Avatar).Equal("http://1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87")
			g.Assert(build.Author).Equal(hook.Sender.Login)

		})

		g.It("Should return a Repo struct from a push hook", func() {
			buf := bytes.NewBufferString(fixtures.HookPush)
			hook, _ := parsePush(buf)
			repo := repoFromPush(hook)
			g.Assert(repo.Name).Equal(hook.Repo.Name)
			g.Assert(repo.Owner).Equal(hook.Repo.Owner.Username)
			g.Assert(repo.FullName).Equal("gordon/hello-world")
			g.Assert(repo.Link).Equal(hook.Repo.URL)
		})

		g.It("Should return a Build struct from a pull_request hook", func() {
			buf := bytes.NewBufferString(fixtures.HookPullRequest)
			hook, _ := parsePullRequest(buf)
			build := buildFromPullRequest(hook)
			g.Assert(build.Event).Equal(model.EventPull)
			g.Assert(build.Commit).Equal(hook.PullRequest.Head.Sha)
			g.Assert(build.Ref).Equal("refs/pull/1/head")
			g.Assert(build.Link).Equal(hook.PullRequest.URL)
			g.Assert(build.Branch).Equal("master")
			g.Assert(build.Message).Equal(hook.PullRequest.Title)
			g.Assert(build.Avatar).Equal("http://1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87")
			g.Assert(build.Author).Equal(hook.PullRequest.User.Username)

		})

		g.It("Should return a Repo struct from a pull_request hook", func() {
			buf := bytes.NewBufferString(fixtures.HookPullRequest)
			hook, _ := parsePullRequest(buf)
			repo := repoFromPullRequest(hook)
			g.Assert(repo.Name).Equal(hook.Repo.Name)
			g.Assert(repo.Owner).Equal(hook.Repo.Owner.Username)
			g.Assert(repo.FullName).Equal("gordon/hello-world")
			g.Assert(repo.Link).Equal(hook.Repo.URL)
		})

		g.It("Should return a Perm struct from a Gogs Perm", func() {
			perms := []gogs.Permission{
				{true, true, true},
				{true, true, false},
				{true, false, false},
			}
			for _, from := range perms {
				perm := toPerm(from)
				g.Assert(perm.Pull).Equal(from.Pull)
				g.Assert(perm.Push).Equal(from.Push)
				g.Assert(perm.Admin).Equal(from.Admin)
			}
		})

		g.It("Should return a Team struct from a Gogs Org", func() {
			from := &gogs.Organization{
				UserName:  "drone",
				AvatarUrl: "/avatars/1",
			}

			to := toTeam(from, "http://localhost:80")
			g.Assert(to.Login).Equal(from.UserName)
			g.Assert(to.Avatar).Equal("http://localhost:80/avatars/1")
		})

		g.It("Should return a Repo struct from a Gogs Repo", func() {
			from := gogs.Repository{
				FullName: "gophers/hello-world",
				Owner: gogs.User{
					UserName:  "gordon",
					AvatarUrl: "http://1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
				},
				CloneUrl: "http://gogs.golang.org/gophers/hello-world.git",
				HtmlUrl:  "http://gogs.golang.org/gophers/hello-world",
				Private:  true,
			}
			repo := toRepo(&from)
			g.Assert(repo.FullName).Equal(from.FullName)
			g.Assert(repo.Owner).Equal(from.Owner.UserName)
			g.Assert(repo.Name).Equal("hello-world")
			g.Assert(repo.Branch).Equal("master")
			g.Assert(repo.Link).Equal(from.HtmlUrl)
			g.Assert(repo.Clone).Equal(from.CloneUrl)
			g.Assert(repo.Avatar).Equal(from.Owner.AvatarUrl)
			g.Assert(repo.IsPrivate).Equal(from.Private)
		})

		g.It("Should return a RepoLite struct from a Gogs Repo", func() {
			from := gogs.Repository{
				FullName: "gophers/hello-world",
				Owner: gogs.User{
					UserName:  "gordon",
					AvatarUrl: "http://1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
				},
			}
			repo := toRepoLite(&from)
			g.Assert(repo.FullName).Equal(from.FullName)
			g.Assert(repo.Owner).Equal(from.Owner.UserName)
			g.Assert(repo.Name).Equal("hello-world")
			g.Assert(repo.Avatar).Equal(from.Owner.AvatarUrl)
		})

		g.It("Should correct a malformed avatar url", func() {

			var urls = []struct {
				Before string
				After  string
			}{
				{
					"http://gogs.golang.org///1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
					"//1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
				},
				{
					"//1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
					"//1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
				},
				{
					"http://gogs.golang.org/avatars/1",
					"http://gogs.golang.org/avatars/1",
				},
				{
					"http://gogs.golang.org//avatars/1",
					"http://gogs.golang.org/avatars/1",
				},
			}

			for _, url := range urls {
				got := fixMalformedAvatar(url.Before)
				g.Assert(got).Equal(url.After)
			}
		})

		g.It("Should expand the avatar url", func() {
			var urls = []struct {
				Before string
				After  string
			}{
				{
					"/avatars/1",
					"http://gogs.io/avatars/1",
				},
				{
					"//1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
					"http://1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
				},
				{
					"/gogs/avatars/2",
					"http://gogs.io/gogs/avatars/2",
				},
			}

			var repo = "http://gogs.io/foo/bar"
			for _, url := range urls {
				got := expandAvatar(repo, url.Before)
				g.Assert(got).Equal(url.After)
			}
		})
	})
}
