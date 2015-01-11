package heroku

import (
	"strings"
	"testing"

	"github.com/drone/drone/shared/build/buildfile"
	"github.com/franela/goblin"
)

func Test_Heroku(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Heroku Deploy", func() {

		g.It("Should set git.config", func() {
			b := new(buildfile.Buildfile)
			h := Heroku{
				App: "drone",
			}

			h.Write(b)
			out := b.String()
			g.Assert(strings.Contains(out, CmdRevParse)).Equal(true)
			g.Assert(strings.Contains(out, CmdGlobalUser)).Equal(true)
			g.Assert(strings.Contains(out, CmdGlobalEmail)).Equal(true)
		})

		g.It("Should write token", func() {
			b := new(buildfile.Buildfile)
			h := Heroku{
				App:   "drone",
				Token: "mock-token",
			}

			h.Write(b)
			out := b.String()
			g.Assert(strings.Contains(out, "\necho 'machine git.heroku.com login _ password mock-token' >> ~/.netrc\n")).Equal(true)
		})

		g.It("Should add remote", func() {
			b := new(buildfile.Buildfile)
			h := Heroku{
				App: "drone",
			}

			h.Write(b)
			out := b.String()
			g.Assert(strings.Contains(out, "\ngit remote add heroku https://git.heroku.com/drone.git\n")).Equal(true)
		})

		g.It("Should push to remote", func() {
			b := new(buildfile.Buildfile)
			d := Heroku{
				App: "drone",
			}

			d.Write(b)
			out := b.String()
			g.Assert(strings.Contains(out, "\ngit push heroku $COMMIT:refs/heads/master\n")).Equal(true)
		})

		g.It("Should force push to remote", func() {
			b := new(buildfile.Buildfile)
			h := Heroku{
				Force: true,
				App:   "drone",
			}

			h.Write(b)
			out := b.String()
			g.Assert(strings.Contains(out, "\ngit add -A\n")).Equal(true)
			g.Assert(strings.Contains(out, "\ngit commit -m 'adding build artifacts'\n")).Equal(true)
			g.Assert(strings.Contains(out, "\ngit push heroku HEAD:refs/heads/master --force\n")).Equal(true)
		})

	})
}
