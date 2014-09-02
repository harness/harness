package github

import (
	"testing"

	"github.com/drone/drone/plugin/remote/github/testdata"
	"github.com/franela/goblin"
)

func Test_Helper(t *testing.T) {
	// setup a dummy github server
	var server = testdata.NewServer()
	defer server.Close()

	g := goblin.Goblin(t)
	g.Describe("GitHub Helper Functions", func() {

		g.It("Should Get a User")
		g.It("Should Get a User Primary Email")
		g.It("Should Get a User + Primary Email")
		g.It("Should Get a list of Orgs")
		g.It("Should Get a list of User Repos")
		g.It("Should Get a list of Org Repos")
		g.It("Should Get a list of All Repos")
		g.It("Should Get a Repo Key")
		g.It("Should Get a Repo Hook")
		g.It("Should Create a Repo Key")
		g.It("Should Create a Repo Hook")
		g.It("Should Create or Update a Repo Key")
		g.It("Should Create or Update a Repo Hook")
		g.It("Should Get a Repo File")

	})
}
