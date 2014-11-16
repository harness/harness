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

		g.It("Should add remote", func() {
			b := new(buildfile.Buildfile)
			h := Heroku{
				App: "drone",
			}

			h.Write(b)
			out := b.String()
			g.Assert(strings.Contains(out, "git remote add heroku git@heroku.com:drone.git")).Equal(true)
		})

		g.It("Should push to remote", func() {
			b := new(buildfile.Buildfile)
			d := Heroku{
				App: "drone",
			}

			d.Write(b)
			out := b.String()
			g.Assert(strings.Contains(out, "git push heroku $COMMIT:master")).Equal(true)
		})

		g.It("Should force push to remote", func() {
			b := new(buildfile.Buildfile)
			h := Heroku{
				Force: true,
				App:   "drone",
			}

			h.Write(b)
			out := b.String()
			g.Assert(strings.Contains(out, "git add -A")).Equal(true)
			g.Assert(strings.Contains(out, "git commit -m 'adding build artifacts'")).Equal(true)
			g.Assert(strings.Contains(out, "git push heroku HEAD:master --force")).Equal(true)
		})

	})
}
