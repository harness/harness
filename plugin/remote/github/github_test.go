package github

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/drone/drone/plugin/remote/github/testdata"
	"github.com/drone/drone/shared/model"
	"github.com/franela/goblin"
)

func Test_Github(t *testing.T) {
	// setup a dummy github server
	var server = testdata.NewServer()
	defer server.Close()

	var github = GitHub{
		URL: server.URL,
		API: server.URL,
	}
	var user = model.User{
		Access: "e3b0c44298fc1c149afbf4c8996fb",
	}
	var repo = model.Repo{
		Owner: "octocat",
		Name:  "Hello-World",
	}
	var hook = model.Hook{
		Sha: "6dcb09b5b57875f334f61aebed695e2e4193db5e",
	}

	g := goblin.Goblin(t)
	g.Describe("GitHub Plugin", func() {

		g.It("Should identify github vs github enterprise", func() {
			var ghc = &GitHub{URL: "https://github.com"}
			var ghe = &GitHub{URL: "https://github.drone.io"}
			g.Assert(ghc.IsEnterprise()).IsFalse()
			g.Assert(ghe.IsEnterprise()).IsTrue()
			g.Assert(ghc.GetKind()).Equal(model.RemoteGithub)
			g.Assert(ghe.GetKind()).Equal(model.RemoteGithubEnterprise)
		})

		g.It("Should parse the hostname", func() {
			var ghc = &GitHub{URL: "https://github.com"}
			var ghe = &GitHub{URL: "https://github.drone.io:80"}
			g.Assert(ghc.GetHost()).Equal("github.com")
			g.Assert(ghe.GetHost()).Equal("github.drone.io:80")
		})

		g.It("Should get the repo list", func() {
			var repos, err = github.GetRepos(&user)
			g.Assert(err == nil).IsTrue()
			g.Assert(len(repos)).Equal(4)
			g.Assert(repos[0].Name).Equal("Hello-World")
			g.Assert(repos[0].Owner).Equal("octocat")
			g.Assert(repos[0].Host).Equal(github.GetHost())
			g.Assert(repos[0].Remote).Equal(github.GetKind())
			g.Assert(repos[0].Private).Equal(true)
			g.Assert(repos[0].CloneURL).Equal("git@github.com:octocat/Hello-World.git")
			g.Assert(repos[0].SSHURL).Equal("git@github.com:octocat/Hello-World.git")
			g.Assert(repos[0].GitURL).Equal("git://github.com/octocat/Hello-World.git")
			g.Assert(repos[0].Role.Admin).Equal(true)
			g.Assert(repos[0].Role.Read).Equal(true)
			g.Assert(repos[0].Role.Write).Equal(true)
		})

		g.It("Should get the build script", func() {
			var script, err = github.GetScript(&user, &repo, &hook)
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

		g.It("Should parse a commit hook")

		g.It("Should parse a pull request hook")

		g.Describe("Authorize", func() {
			g.AfterEach(func() {
				github.Orgs = []string{}
			})

			var resp = httptest.NewRecorder()
			var state = "validstate"
			var req, _ = http.NewRequest(
				"GET",
				fmt.Sprintf("%s/?code=sekret&state=%s", server.URL, state),
				nil,
			)
			req.AddCookie(&http.Cookie{Name: "github_state", Value: state})

			g.It("Should authorize a valid user with no org restrictions", func() {
				var login, err = github.Authorize(resp, req)
				g.Assert(err == nil).IsTrue()
				g.Assert(login == nil).IsFalse()
			})

			g.It("Should authorize a valid user in the correct org", func() {
				github.Orgs = []string{"octocats-inc"}
				var login, err = github.Authorize(resp, req)
				g.Assert(err == nil).IsTrue()
				g.Assert(login == nil).IsFalse()
			})

			g.It("Should not authorize a valid user in the wrong org", func() {
				github.Orgs = []string{"acme"}
				var login, err = github.Authorize(resp, req)
				g.Assert(err != nil).IsTrue()
				g.Assert(login == nil).IsTrue()
			})
		})
	})
}
