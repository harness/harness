package modulus

import (
	"testing"

	"github.com/drone/drone/shared/build/buildfile"
	"github.com/franela/goblin"
)

func Test_Modulus(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Modulus Deploy", func() {

		g.It("Requires a Project name", func() {
			b := new(buildfile.Buildfile)
			m := Modulus{
				Project: "foo",
			}

			m.Write(b)
			g.Assert(b.String()).Equal("")
		})

		g.It("Requires a Token", func() {
			b := new(buildfile.Buildfile)
			m := Modulus{
				Token: "bar",
			}

			m.Write(b)
			g.Assert(b.String()).Equal("")
		})

		g.It("Should execute deploy commands", func() {
			b := new(buildfile.Buildfile)
			m := Modulus{
				Project: "foo",
				Token:   "bar",
			}

			m.Write(b)
			g.Assert(b.String()).Equal(`export MODULUS_TOKEN="bar"
sh -c "[ -f /usr/bin/npm ] || echo ERROR: npm is required for modulus.io deployments"
sh -c "[ -f /usr/bin/npm ] || exit 1"
sh -c "[ -f /usr/bin/sudo ] || npm install -g modulus"
sh -c "[ -f /usr/bin/sudo ] && sudo npm install -g modulus"
echo '#DRONE:6d6f64756c7573206465706c6f79202d702022666f6f22'
sh -c "modulus deploy -p \"foo\""
`)
		})
	})
}
