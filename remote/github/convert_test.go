package github

import (
	"testing"

	"github.com/drone/drone/model"
	"github.com/google/go-github/github"

	"github.com/franela/goblin"
)

func Test_helper(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("GitHub converter", func() {

		g.It("should convert passing status", func() {
			g.Assert(convertStatus(model.StatusSuccess)).Equal(statusSuccess)
		})

		g.It("should convert pending status", func() {
			g.Assert(convertStatus(model.StatusPending)).Equal(statusPending)
			g.Assert(convertStatus(model.StatusRunning)).Equal(statusPending)
		})

		g.It("should convert failing status", func() {
			g.Assert(convertStatus(model.StatusFailure)).Equal(statusFailure)
		})

		g.It("should convert error status", func() {
			g.Assert(convertStatus(model.StatusKilled)).Equal(statusError)
			g.Assert(convertStatus(model.StatusError)).Equal(statusError)
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
			from := github.Repository{
				FullName: github.String("octocat/hello-world"),
				Name:     github.String("hello-world"),
				Owner: &github.User{
					AvatarURL: github.String("http://..."),
					Login:     github.String("octocat"),
				},
			}

			to := convertRepoLite(from)
			g.Assert(to.Avatar).Equal("http://...")
			g.Assert(to.FullName).Equal("octocat/hello-world")
			g.Assert(to.Owner).Equal("octocat")
			g.Assert(to.Name).Equal("hello-world")
		})

		g.It("should convert repository list", func() {
			from := []github.Repository{
				{
					FullName: github.String("octocat/hello-world"),
					Name:     github.String("hello-world"),
					Owner: &github.User{
						AvatarURL: github.String("http://..."),
						Login:     github.String("octocat"),
					},
				},
			}

			to := convertRepoList(from)
			g.Assert(to[0].Avatar).Equal("http://...")
			g.Assert(to[0].FullName).Equal("octocat/hello-world")
			g.Assert(to[0].Owner).Equal("octocat")
			g.Assert(to[0].Name).Equal("hello-world")
		})

		g.It("should convert repository", func() {
			from := github.Repository{
				FullName:      github.String("octocat/hello-world"),
				Name:          github.String("hello-world"),
				HTMLURL:       github.String("https://github.com/octocat/hello-world"),
				CloneURL:      github.String("https://github.com/octocat/hello-world.git"),
				DefaultBranch: github.String("develop"),
				Private:       github.Bool(true),
				Owner: &github.User{
					AvatarURL: github.String("http://..."),
					Login:     github.String("octocat"),
				},
			}

			to := convertRepo(&from, false)
			g.Assert(to.Avatar).Equal("http://...")
			g.Assert(to.FullName).Equal("octocat/hello-world")
			g.Assert(to.Owner).Equal("octocat")
			g.Assert(to.Name).Equal("hello-world")
			g.Assert(to.Branch).Equal("develop")
			g.Assert(to.Kind).Equal("git")
			g.Assert(to.IsPrivate).IsTrue()
			g.Assert(to.Clone).Equal("https://github.com/octocat/hello-world.git")
			g.Assert(to.Link).Equal("https://github.com/octocat/hello-world")
		})

		g.It("should convert repository permissions", func() {
			from := &github.Repository{
				Permissions: &map[string]bool{
					"admin": true,
					"push":  true,
					"pull":  true,
				},
			}

			to := convertPerm(from)
			g.Assert(to.Push).IsTrue()
			g.Assert(to.Pull).IsTrue()
			g.Assert(to.Admin).IsTrue()
		})

		g.It("should convert team", func() {
			from := github.Organization{
				Login:     github.String("octocat"),
				AvatarURL: github.String("http://..."),
			}
			to := convertTeam(from)
			g.Assert(to.Login).Equal("octocat")
			g.Assert(to.Avatar).Equal("http://...")
		})

		g.It("should convert team list", func() {
			from := []github.Organization{
				{
					Login:     github.String("octocat"),
					AvatarURL: github.String("http://..."),
				},
			}
			to := convertTeamList(from)
			g.Assert(to[0].Login).Equal("octocat")
			g.Assert(to[0].Avatar).Equal("http://...")
		})

		//
		// g.It("should convert user", func() {
		// 	token := &oauth2.Token{
		// 		AccessToken:  "foo",
		// 		RefreshToken: "bar",
		// 		Expiry:       time.Now(),
		// 	}
		// 	user := &internal.Account{Login: "octocat"}
		// 	user.Links.Avatar.Href = "http://..."
		//
		// 	result := convertUser(user, token)
		// 	g.Assert(result.Avatar).Equal(user.Links.Avatar.Href)
		// 	g.Assert(result.Login).Equal(user.Login)
		// 	g.Assert(result.Token).Equal(token.AccessToken)
		// 	g.Assert(result.Token).Equal(token.AccessToken)
		// 	g.Assert(result.Secret).Equal(token.RefreshToken)
		// 	g.Assert(result.Expiry).Equal(token.Expiry.UTC().Unix())
		// })
	})
}
