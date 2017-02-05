package bitbucket

import (
	"testing"
	"time"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote/bitbucket/internal"

	"github.com/franela/goblin"
	"golang.org/x/oauth2"
)

func Test_helper(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Bitbucket converter", func() {

		g.It("should convert passing status", func() {
			g.Assert(convertStatus(model.StatusSuccess)).Equal(statusSuccess)
		})

		g.It("should convert pending status", func() {
			g.Assert(convertStatus(model.StatusPending)).Equal(statusPending)
			g.Assert(convertStatus(model.StatusRunning)).Equal(statusPending)
		})

		g.It("should convert failing status", func() {
			g.Assert(convertStatus(model.StatusFailure)).Equal(statusFailure)
			g.Assert(convertStatus(model.StatusKilled)).Equal(statusFailure)
			g.Assert(convertStatus(model.StatusError)).Equal(statusFailure)
		})

		g.It("should convert passing desc", func() {
			g.Assert(convertDesc(model.StatusSuccess)).Equal(descSuccess)
		})

		g.It("should convert pending desc", func() {
			g.Assert(convertDesc(model.StatusPending)).Equal(descPending)
			g.Assert(convertDesc(model.StatusRunning)).Equal(descPending)
		})

		g.It("should convert failing desc", func() {
			g.Assert(convertDesc(model.StatusFailure)).Equal(descFailure)
		})

		g.It("should convert error desc", func() {
			g.Assert(convertDesc(model.StatusKilled)).Equal(descError)
			g.Assert(convertDesc(model.StatusError)).Equal(descError)
		})

		g.It("should convert repository lite", func() {
			from := &internal.Repo{}
			from.FullName = "octocat/hello-world"
			from.Owner.Links.Avatar.Href = "http://..."

			to := convertRepoLite(from)
			g.Assert(to.Avatar).Equal(from.Owner.Links.Avatar.Href)
			g.Assert(to.FullName).Equal(from.FullName)
			g.Assert(to.Owner).Equal("octocat")
			g.Assert(to.Name).Equal("hello-world")
		})

		g.It("should convert repository", func() {
			from := &internal.Repo{
				FullName:  "octocat/hello-world",
				IsPrivate: true,
				Scm:       "hg",
			}
			from.Owner.Links.Avatar.Href = "http://..."
			from.Links.Html.Href = "https://bitbucket.org/foo/bar"

			to := convertRepo(from)
			g.Assert(to.Avatar).Equal(from.Owner.Links.Avatar.Href)
			g.Assert(to.FullName).Equal(from.FullName)
			g.Assert(to.Owner).Equal("octocat")
			g.Assert(to.Name).Equal("hello-world")
			g.Assert(to.Branch).Equal("default")
			g.Assert(to.Kind).Equal(from.Scm)
			g.Assert(to.IsPrivate).Equal(from.IsPrivate)
			g.Assert(to.Clone).Equal(from.Links.Html.Href)
			g.Assert(to.Link).Equal(from.Links.Html.Href)
		})

		g.It("should convert team", func() {
			from := &internal.Account{Login: "octocat"}
			from.Links.Avatar.Href = "http://..."
			to := convertTeam(from)
			g.Assert(to.Avatar).Equal(from.Links.Avatar.Href)
			g.Assert(to.Login).Equal(from.Login)
		})

		g.It("should convert team list", func() {
			from := &internal.Account{Login: "octocat"}
			from.Links.Avatar.Href = "http://..."
			to := convertTeamList([]*internal.Account{from})
			g.Assert(to[0].Avatar).Equal(from.Links.Avatar.Href)
			g.Assert(to[0].Login).Equal(from.Login)
		})

		g.It("should convert user", func() {
			token := &oauth2.Token{
				AccessToken:  "foo",
				RefreshToken: "bar",
				Expiry:       time.Now(),
			}
			user := &internal.Account{Login: "octocat"}
			user.Links.Avatar.Href = "http://..."

			result := convertUser(user, token)
			g.Assert(result.Avatar).Equal(user.Links.Avatar.Href)
			g.Assert(result.Login).Equal(user.Login)
			g.Assert(result.Token).Equal(token.AccessToken)
			g.Assert(result.Secret).Equal(token.RefreshToken)
			g.Assert(result.Expiry).Equal(token.Expiry.UTC().Unix())
		})

		g.It("should use clone url", func() {
			repo := &internal.Repo{}
			repo.Links.Clone = append(repo.Links.Clone, internal.Link{
				Name: "https",
				Href: "https://bitbucket.org/foo/bar.git",
			})
			link := cloneLink(repo)
			g.Assert(link).Equal(repo.Links.Clone[0].Href)
		})

		g.It("should build clone url", func() {
			repo := &internal.Repo{}
			repo.Links.Html.Href = "https://foo:bar@bitbucket.org/foo/bar.git"
			link := cloneLink(repo)
			g.Assert(link).Equal("https://bitbucket.org/foo/bar.git")
		})

		g.It("should convert pull hook to build", func() {
			hook := &internal.PullRequestHook{}
			hook.Actor.Login = "octocat"
			hook.Actor.Links.Avatar.Href = "https://..."
			hook.PullRequest.Dest.Commit.Hash = "73f9c44d"
			hook.PullRequest.Dest.Branch.Name = "master"
			hook.PullRequest.Dest.Repo.Links.Html.Href = "https://bitbucket.org/foo/bar"
			hook.PullRequest.Source.Branch.Name = "change"
			hook.PullRequest.Source.Repo.FullName = "baz/bar"
			hook.PullRequest.Links.Html.Href = "https://bitbucket.org/foo/bar/pulls/5"
			hook.PullRequest.Desc = "updated README"
			hook.PullRequest.Updated = time.Now()

			build := convertPullHook(hook)
			g.Assert(build.Event).Equal(model.EventPull)
			g.Assert(build.Author).Equal(hook.Actor.Login)
			g.Assert(build.Avatar).Equal(hook.Actor.Links.Avatar.Href)
			g.Assert(build.Commit).Equal(hook.PullRequest.Dest.Commit.Hash)
			g.Assert(build.Branch).Equal(hook.PullRequest.Dest.Branch.Name)
			g.Assert(build.Link).Equal(hook.PullRequest.Links.Html.Href)
			g.Assert(build.Ref).Equal("refs/heads/master")
			g.Assert(build.Refspec).Equal("change:master")
			g.Assert(build.Remote).Equal("https://bitbucket.org/baz/bar")
			g.Assert(build.Message).Equal(hook.PullRequest.Desc)
			g.Assert(build.Timestamp).Equal(hook.PullRequest.Updated.Unix())
		})

		g.It("should convert push hook to build", func() {
			change := internal.Change{}
			change.New.Target.Hash = "73f9c44d"
			change.New.Name = "master"
			change.New.Target.Links.Html.Href = "https://bitbucket.org/foo/bar/commits/73f9c44d"
			change.New.Target.Message = "updated README"
			change.New.Target.Date = time.Now()
			change.New.Target.Author.Raw = "Test <test@domain.tld>"

			hook := internal.PushHook{}
			hook.Actor.Login = "octocat"
			hook.Actor.Links.Avatar.Href = "https://..."

			build := convertPushHook(&hook, &change)
			g.Assert(build.Event).Equal(model.EventPush)
			g.Assert(build.Email).Equal("test@domain.tld")
			g.Assert(build.Author).Equal(hook.Actor.Login)
			g.Assert(build.Avatar).Equal(hook.Actor.Links.Avatar.Href)
			g.Assert(build.Commit).Equal(change.New.Target.Hash)
			g.Assert(build.Branch).Equal(change.New.Name)
			g.Assert(build.Link).Equal(change.New.Target.Links.Html.Href)
			g.Assert(build.Ref).Equal("refs/heads/master")
			g.Assert(build.Message).Equal(change.New.Target.Message)
			g.Assert(build.Timestamp).Equal(change.New.Target.Date.Unix())
		})

		g.It("should convert tag hook to build", func() {
			change := internal.Change{}
			change.New.Name = "v1.0.0"
			change.New.Type = "tag"

			hook := internal.PushHook{}

			build := convertPushHook(&hook, &change)
			g.Assert(build.Event).Equal(model.EventTag)
			g.Assert(build.Ref).Equal("refs/tags/v1.0.0")
		})
	})
}
