package bitbucketserver

import (
	"github.com/drone/drone/model"
	"github.com/drone/drone/remote/bitbucketserver/internal"
	"github.com/franela/goblin"
	"github.com/mrjones/oauth"
	"testing"
)

func Test_helper(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Bitbucket Server converter", func() {

		g.It("should convert repository lite", func() {
			from := &internal.Repo{}
			from.Project.Key = "octocat"
			from.Slug = "hello-world"

			to := convertRepoLite(from)
			g.Assert(to.FullName).Equal("octocat/hello-world")
			g.Assert(to.Owner).Equal("octocat")
			g.Assert(to.Name).Equal("hello-world")
		})

		g.It("should convert repository", func() {
			from := &internal.Repo{
				Slug: "hello-world",
			}
			from.Project.Key = "octocat"

			//var links [1]internal.LinkType
			link := internal.CloneLink{
				Name: "http",
				Href: "https://x7hw@server.org/foo/bar.git",
			}
			from.Links.Clone = append(from.Links.Clone, link)

			selfRef := internal.SelfRefLink{
				Href: "https://server.org/foo/bar",
			}

			from.Links.Self = append(from.Links.Self, selfRef)

			to := convertRepo(from)
			g.Assert(to.FullName).Equal("octocat/hello-world")
			g.Assert(to.Owner).Equal("octocat")
			g.Assert(to.Name).Equal("hello-world")
			g.Assert(to.Branch).Equal("master")
			g.Assert(to.Kind).Equal(model.RepoGit)
			g.Assert(to.IsPrivate).Equal(true)
			g.Assert(to.Clone).Equal("https://server.org/foo/bar.git")
			g.Assert(to.Link).Equal("https://server.org/foo/bar")
		})

		g.It("should convert user", func() {
			token := &oauth.AccessToken{
				Token: "foo",
			}
			user := &internal.User{
				Slug:         "x12f",
				EmailAddress: "huh@huh.com",
			}

			result := convertUser(user, token)
			g.Assert(result.Avatar).Equal(avatarLink("huh@huh.com"))
			g.Assert(result.Login).Equal("x12f")
			g.Assert(result.Token).Equal("foo")
		})

		g.It("branch should be empty", func() {
			change := internal.PostHook{}
			change.RefChanges = append(change.RefChanges, internal.RefChange{
				RefID:  "refs/heads/",
				ToHash: "73f9c44d",
			})

			value := internal.Value{}
			value.ToCommit.Author.Name = "John Doe, Appleboy, Mary, Janet E. Dawson and Ann S. Palmer"
			value.ToCommit.Author.EmailAddress = "huh@huh.com"
			value.ToCommit.Message = "message"

			change.Changesets.Values = append(change.Changesets.Values, value)

			change.Repository.Project.Key = "octocat"
			change.Repository.Slug = "hello-world"
			build := convertPushHook(&change, "http://base.com")
			g.Assert(build.Branch).Equal("")
		})

		g.It("should convert push hook to build", func() {
			change := internal.PostHook{}

			change.RefChanges = append(change.RefChanges, internal.RefChange{
				RefID:  "refs/heads/release/some-feature",
				ToHash: "73f9c44d",
			})

			value := internal.Value{}
			value.ToCommit.Author.Name = "John Doe, Appleboy, Mary, Janet E. Dawson and Ann S. Palmer"
			value.ToCommit.Author.EmailAddress = "huh@huh.com"
			value.ToCommit.Message = "message"

			change.Changesets.Values = append(change.Changesets.Values, value)

			change.Repository.Project.Key = "octocat"
			change.Repository.Slug = "hello-world"

			build := convertPushHook(&change, "http://base.com")
			g.Assert(build.Event).Equal(model.EventPush)
			// Ensuring the author label is not longer then 40
			g.Assert(build.Author).Equal("John Doe, Appleboy, Mary, Janet E. Da...")
			g.Assert(build.Avatar).Equal(avatarLink("huh@huh.com"))
			g.Assert(build.Commit).Equal("73f9c44d")
			g.Assert(build.Branch).Equal("release/some-feature")
			g.Assert(build.Link).Equal("http://base.com/projects/octocat/repos/hello-world/commits/73f9c44d")
			g.Assert(build.Ref).Equal("refs/heads/release/some-feature")
			g.Assert(build.Message).Equal("message")
		})

		g.It("should convert tag hook to build", func() {
			change := internal.PostHook{}
			change.RefChanges = append(change.RefChanges, internal.RefChange{
				RefID:  "refs/tags/v1",
				ToHash: "73f9c44d",
			})

			value := internal.Value{}
			value.ToCommit.Author.Name = "John Doe"
			value.ToCommit.Author.EmailAddress = "huh@huh.com"
			value.ToCommit.Message = "message"

			change.Changesets.Values = append(change.Changesets.Values, value)
			change.Repository.Project.Key = "octocat"
			change.Repository.Slug = "hello-world"

			build := convertPushHook(&change, "http://base.com")
			g.Assert(build.Event).Equal(model.EventTag)
			g.Assert(build.Author).Equal("John Doe")
			g.Assert(build.Avatar).Equal(avatarLink("huh@huh.com"))
			g.Assert(build.Commit).Equal("73f9c44d")
			g.Assert(build.Branch).Equal("v1")
			g.Assert(build.Link).Equal("http://base.com/projects/octocat/repos/hello-world/commits/73f9c44d")
			g.Assert(build.Ref).Equal("refs/tags/v1")
			g.Assert(build.Message).Equal("message")
		})
	})
}
