package npm

import (
	"testing"

	"github.com/franela/goblin"
)

func Test_NPM(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("NPM Publish", func() {
		g.It("Should set force")
		g.It("Should set tag")
		g.It("Should set registry")
		g.It("Should set always-auth")
		g.It("Should run publish")
		g.It("Should create npmrc")
		g.It("Should fail when no username or password")
		g.It("Should use default username or password")
	})
}
