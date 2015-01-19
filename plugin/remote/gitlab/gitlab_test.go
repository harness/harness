package gitlab

import (
	"net/http"
	"testing"

	"github.com/drone/drone/plugin/remote/gitlab/testdata"
	"github.com/drone/drone/shared/model"
	"github.com/franela/goblin"
)

func Test_Github(t *testing.T) {
	// setup a dummy github server
	var server = testdata.NewServer()
	defer server.Close()

	var gitlab = New(server.URL, false, false)
	var user = model.User{
		Access: "e3b0c44298fc1c149afbf4c8996fb",
	}
	/*
		var repo = model.Repo{
			Owner: "gitlab",
			Name:  "Hello-World",
		}
		var commit = model.Commit{
			Sha: "6dcb09b5b57875f334f61aebed695e2e4193db5e",
		}
	*/

	g := goblin.Goblin(t)
	g.Describe("Gitlab Plugin", func() {

		g.It("Should authorize user", func() {
			var req, _ = http.NewRequest("GET", "/login/gitlab", nil)
			var login, err = gitlab.Authorize(nil, req)
			g.Assert(err == nil).IsTrue()
			g.Assert(login.Email).Equal("john@example.com")
			g.Assert(login.Name).Equal("John Smith")
			g.Assert(login.Login).Equal("john_smith")
			g.Assert(login.Access).Equal("dd34asd13as")
			g.Assert(login.ID).Equal(int64(1))
		})

		g.It("Should get the repo list", func() {
			var repos, err = gitlab.GetRepos(&user)
			g.Assert(err == nil).IsTrue()
			g.Assert(len(repos)).Equal(2)
			g.Assert(repos[0].Name).Equal("diaspora-client")
			g.Assert(repos[0].Owner).Equal("diaspora")
			g.Assert(repos[0].Host).Equal(gitlab.GetHost())
			g.Assert(repos[0].Remote).Equal(gitlab.GetKind())
			g.Assert(repos[0].Private).Equal(true)
			g.Assert(repos[0].Role.Admin).Equal(true)
			g.Assert(repos[0].Role.Read).Equal(true)
			g.Assert(repos[0].Role.Write).Equal(true)
		})
		/*
			g.It("Should get the build script", func() {
				var script, err = github.GetScript(&user, &repo, &commit)
				g.Assert(err == nil).IsTrue()
				g.Assert(string(script)).Equal("image: go")
			})

			g.It("Should activate a public repo", func() {
				repo.Private = false
				repo.CloneURL = "git://github.com/octocat/Hello-World.git"
				repo.SSHURL = "git@github.com:octocat/Hello-World.git"
				var err = github.Activate(&user, &repo, "http://example.com")
				g.Assert(err == nil).IsTrue()
			})

			g.It("Should activate a private repo", func() {
				repo.Name = "Hola-Mundo"
				repo.Private = true
				repo.CloneURL = "git@github.com:octocat/Hola-Mundo.git"
				repo.SSHURL = "git@github.com:octocat/Hola-Mundo.git"
				var err = github.Activate(&user, &repo, "http://example.com")
				g.Assert(err == nil).IsTrue()
			})
		*/
		g.It("Should parse a commit hook")

		g.It("Should ignore a pull request hook")
	})
}
