package tsuru

import (
	"strings"
	"testing"

	"github.com/drone/drone/shared/build/buildfile"
	"github.com/franela/goblin"
)

func Test_Tsuru(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Tsuru Deploy", func() {

		g.It("Should set git.config", func() {
			b := new(buildfile.Buildfile)
			d := Tsuru{
				Remote: "git://foo.com/bar/baz.git",
			}

			d.Write(b)
			out := b.String()
			g.Assert(strings.Contains(out, CmdRevParse)).Equal(true)
			g.Assert(strings.Contains(out, CmdGlobalUser)).Equal(true)
			g.Assert(strings.Contains(out, CmdGlobalEmail)).Equal(true)
		})

		g.It("Should add remote", func() {
			b := new(buildfile.Buildfile)
			d := Tsuru{
				Remote: "git://foo.com/bar/baz.git",
			}

			d.Write(b)
			out := b.String()
			g.Assert(strings.Contains(out, "git remote add tsuru git://foo.com/bar/baz.git")).Equal(true)
		})

		g.It("Should push to remote", func() {
			b := new(buildfile.Buildfile)
			d := Tsuru{
				Remote: "git://foo.com/bar/baz.git",
			}

			d.Write(b)
			out := b.String()
			g.Assert(strings.Contains(out, "git push tsuru $COMMIT:master")).Equal(true)
		})

		g.It("Should force push to remote", func() {
			b := new(buildfile.Buildfile)
			d := Tsuru{
				Force:  true,
				Remote: "git://foo.com/bar/baz.git",
			}

			d.Write(b)
			out := b.String()
			g.Assert(strings.Contains(out, "git add -A")).Equal(true)
			g.Assert(strings.Contains(out, "git commit -m 'adding build artifacts'")).Equal(true)
			g.Assert(strings.Contains(out, "git push tsuru HEAD:master --force")).Equal(true)
		})

	})
}
