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

		g.It("should convert a repository from webhook", func() {
			from := &webhook{}
			from.Repo.Owner.Login = "octocat"
			from.Repo.Owner.Name = "octocat"
			from.Repo.Name = "hello-world"
			from.Repo.FullName = "octocat/hello-world"
			from.Repo.Private = true
			from.Repo.HTMLURL = "https://github.com/octocat/hello-world"
			from.Repo.CloneURL = "https://github.com/octocat/hello-world.git"
			from.Repo.DefaultBranch = "develop"

			repo := convertRepoHook(from)
			g.Assert(repo.Owner).Equal(from.Repo.Owner.Login)
			g.Assert(repo.Name).Equal(from.Repo.Name)
			g.Assert(repo.FullName).Equal(from.Repo.FullName)
			g.Assert(repo.IsPrivate).Equal(from.Repo.Private)
			g.Assert(repo.Link).Equal(from.Repo.HTMLURL)
			g.Assert(repo.Clone).Equal(from.Repo.CloneURL)
			g.Assert(repo.Branch).Equal(from.Repo.DefaultBranch)
		})

		g.It("should convert a pull request from webhook", func() {
			from := &webhook{}
			from.PullRequest.Base.Ref = "master"
			from.PullRequest.Head.Ref = "changes"
			from.PullRequest.Head.SHA = "f72fc19"
			from.PullRequest.Head.Repo.CloneURL = "https://github.com/octocat/hello-world-fork"
			from.PullRequest.HTMLURL = "https://github.com/octocat/hello-world/pulls/42"
			from.PullRequest.Number = 42
			from.PullRequest.Title = "Updated README.md"
			from.PullRequest.User.Login = "octocat"
			from.PullRequest.User.Avatar = "https://avatars1.githubusercontent.com/u/583231"
			from.Sender.Login = "octocat"

			build := convertPullHook(from, true)
			g.Assert(build.Event).Equal(model.EventPull)
			g.Assert(build.Branch).Equal(from.PullRequest.Base.Ref)
			g.Assert(build.Ref).Equal("refs/pull/42/merge")
			g.Assert(build.Refspec).Equal("changes:master")
			g.Assert(build.Remote).Equal("https://github.com/octocat/hello-world-fork")
			g.Assert(build.Commit).Equal(from.PullRequest.Head.SHA)
			g.Assert(build.Message).Equal(from.PullRequest.Title)
			g.Assert(build.Title).Equal(from.PullRequest.Title)
			g.Assert(build.Author).Equal(from.PullRequest.User.Login)
			g.Assert(build.Avatar).Equal(from.PullRequest.User.Avatar)
			g.Assert(build.Sender).Equal(from.Sender.Login)
		})

		g.It("should convert a deployment from webhook", func() {
			from := &webhook{}
			from.Deployment.Desc = ":shipit:"
			from.Deployment.Env = "production"
			from.Deployment.ID = 42
			from.Deployment.Ref = "master"
			from.Deployment.Sha = "f72fc19"
			from.Deployment.URL = "https://github.com/octocat/hello-world"
			from.Sender.Login = "octocat"
			from.Sender.Avatar = "https://avatars1.githubusercontent.com/u/583231"

			build := convertDeployHook(from)
			g.Assert(build.Event).Equal(model.EventDeploy)
			g.Assert(build.Branch).Equal("master")
			g.Assert(build.Ref).Equal("refs/heads/master")
			g.Assert(build.Commit).Equal(from.Deployment.Sha)
			g.Assert(build.Message).Equal(from.Deployment.Desc)
			g.Assert(build.Link).Equal(from.Deployment.URL)
			g.Assert(build.Author).Equal(from.Sender.Login)
			g.Assert(build.Avatar).Equal(from.Sender.Avatar)
		})

		g.It("should convert a push from webhook", func() {
			from := &webhook{}
			from.Sender.Login = "octocat"
			from.Sender.Avatar = "https://avatars1.githubusercontent.com/u/583231"
			from.Repo.CloneURL = "https://github.com/octocat/hello-world.git"
			from.Head.Author.Email = "octocat@github.com"
			from.Head.Message = "updated README.md"
			from.Head.URL = "https://github.com/octocat/hello-world"
			from.Head.ID = "f72fc19"
			from.Ref = "refs/heads/master"

			build := convertPushHook(from)
			g.Assert(build.Event).Equal(model.EventPush)
			g.Assert(build.Branch).Equal("master")
			g.Assert(build.Ref).Equal("refs/heads/master")
			g.Assert(build.Commit).Equal(from.Head.ID)
			g.Assert(build.Message).Equal(from.Head.Message)
			g.Assert(build.Link).Equal(from.Head.URL)
			g.Assert(build.Author).Equal(from.Sender.Login)
			g.Assert(build.Avatar).Equal(from.Sender.Avatar)
			g.Assert(build.Email).Equal(from.Head.Author.Email)
			g.Assert(build.Remote).Equal(from.Repo.CloneURL)
		})

		g.It("should convert a tag from webhook", func() {
			from := &webhook{}
			from.Ref = "refs/tags/v1.0.0"

			build := convertPushHook(from)
			g.Assert(build.Event).Equal(model.EventTag)
			g.Assert(build.Ref).Equal("refs/tags/v1.0.0")
		})
	})
}
