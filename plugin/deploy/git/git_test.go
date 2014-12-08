package git

import (
	"strings"
	"testing"

	"github.com/drone/drone/shared/build/buildfile"
	"github.com/franela/goblin"
)

func Test_Git(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Git Deploy", func() {

		g.It("Should set git.config", func() {
			b := new(buildfile.Buildfile)
			d := Git{
				Target: "git://foo.com/bar/baz.git",
			}

			d.Write(b)
			out := b.String()
			g.Assert(strings.Contains(out, CmdRevParse)).Equal(true)
			g.Assert(strings.Contains(out, CmdGlobalUser)).Equal(true)
			g.Assert(strings.Contains(out, CmdGlobalEmail)).Equal(true)
		})

		g.It("Should add remote", func() {
			b := new(buildfile.Buildfile)
			d := Git{
				Target: "git://foo.com/bar/baz.git",
			}

			d.Write(b)
			out := b.String()
			g.Assert(strings.Contains(out, "git remote add deploy git://foo.com/bar/baz.git")).Equal(true)
		})

		g.It("Should push to remote", func() {
			b := new(buildfile.Buildfile)
			d := Git{
				Target: "git://foo.com/bar/baz.git",
			}

			d.Write(b)
			out := b.String()
			g.Assert(strings.Contains(out, "git push deploy $COMMIT:master")).Equal(true)
		})

		g.It("Should push to alternate branch", func() {
			b := new(buildfile.Buildfile)
			d := Git{
				Branch: "foo",
				Target: "git://foo.com/bar/baz.git",
			}

			d.Write(b)
			out := b.String()
			g.Assert(strings.Contains(out, "git push deploy $COMMIT:foo")).Equal(true)
		})

		g.It("Should force push to remote", func() {
			b := new(buildfile.Buildfile)
			d := Git{
				Force:  true,
				Target: "git://foo.com/bar/baz.git",
			}

			d.Write(b)
			out := b.String()
			g.Assert(strings.Contains(out, "git add -A")).Equal(true)
			g.Assert(strings.Contains(out, "git commit -m 'add build artifacts'")).Equal(true)
			g.Assert(strings.Contains(out, "git push deploy HEAD:master --force")).Equal(true)
		})

	})
}
