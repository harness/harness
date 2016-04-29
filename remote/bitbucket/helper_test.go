package bitbucket

import (
	"testing"
	"time"

	"github.com/drone/drone/remote/bitbucket/internal"

	"github.com/franela/goblin"
	"golang.org/x/oauth2"
)

func Test_helper(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Bitbucket", func() {

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
	})
}
