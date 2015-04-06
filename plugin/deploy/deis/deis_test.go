package deis

import (
	"strings"
	"testing"

	"github.com/drone/drone/shared/build/buildfile"
	"github.com/franela/goblin"
)

func Test_Deis(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Deis Deploy", func() {

		g.It("Should set git.config", func() {
			b := new(buildfile.Buildfile)
			h := Deis{
				App:     "drone",
				Deisurl: "deis.yourdomain.com:2222",
			}

			h.Write(b)
			out := b.String()
			g.Assert(strings.Contains(out, CmdRevParse)).Equal(true)
			g.Assert(strings.Contains(out, CmdGlobalUser)).Equal(true)
			g.Assert(strings.Contains(out, CmdGlobalEmail)).Equal(true)
		})

		g.It("Should add remote", func() {
			b := new(buildfile.Buildfile)
			h := Deis{
				App:     "drone",
				Deisurl: "deis.yourdomain.com:2222/",
			}

			h.Write(b)
			out := b.String()
			g.Assert(strings.Contains(out, "\ngit remote add deis ssh://git@deis.yourdomain.com:2222/drone.git\n")).Equal(true)
		})

		g.It("Should push to remote", func() {
			b := new(buildfile.Buildfile)
			d := Deis{
				App: "drone",
			}

			d.Write(b)
			out := b.String()
			g.Assert(strings.Contains(out, "\ngit push deis $COMMIT:master\n")).Equal(true)
		})

		g.It("Should force push to remote", func() {
			b := new(buildfile.Buildfile)
			h := Deis{
				Force: true,
				App:   "drone",
			}

			h.Write(b)
			out := b.String()
			g.Assert(strings.Contains(out, "\ngit add -A\n")).Equal(true)
			g.Assert(strings.Contains(out, "\ngit commit -m 'adding build artifacts'\n")).Equal(true)
			g.Assert(strings.Contains(out, "\ngit push deis HEAD:master --force\n")).Equal(true)
		})

	})
}
