package nodejitsu

import (
	"testing"

	"github.com/drone/drone/shared/build/buildfile"
	"github.com/franela/goblin"
)

func Test_Modulus(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Nodejitsu Deploy", func() {

		g.It("Requires a User", func() {
			b := new(buildfile.Buildfile)
			n := Nodejitsu{
				User: "foo",
			}

			n.Write(b)
			g.Assert(b.String()).Equal("")
		})

		g.It("Requires a Token", func() {
			b := new(buildfile.Buildfile)
			n := Nodejitsu{
				Token: "bar",
			}

			n.Write(b)
			g.Assert(b.String()).Equal("")
		})

		g.It("Should execute deploy commands", func() {
			b := new(buildfile.Buildfile)
			n := Nodejitsu{
				User:  "foo",
				Token: "bar",
			}

			n.Write(b)
			g.Assert(b.String()).Equal(`export username="foo"
export apiToken="bar"
[ -f /usr/bin/sudo ] || npm install -g jitsu
[ -f /usr/bin/sudo ] && sudo npm install -g jitsu
echo '#DRONE:6a69747375206465706c6f79'
jitsu deploy
`)
		})
	})
}
