package npm

import (
	"strings"
	"testing"

	"github.com/drone/drone/shared/build/buildfile"
	"github.com/franela/goblin"
)

func Test_NPM(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("NPM Publish", func() {

		g.BeforeEach(func() {
			var user, pass, email = "", "", ""
			DefaultEmail = &user
			DefaultUser = &pass
			DefaultPass = &email
		})

		g.It("Should run publish", func() {
			b := new(buildfile.Buildfile)
			n := NPM{
				Email:    "foo@bar.com",
				Username: "foo",
				Password: "bar",
				Folder:   "/path/to/repo",
			}

			n.Write(b)
			out := b.String()
			g.Assert(strings.Contains(out, "npm publish /path/to/repo")).Equal(true)
			g.Assert(strings.Contains(out, "npm set")).Equal(false)
			g.Assert(strings.Contains(out, "npm config set")).Equal(false)
		})

		g.It("Should set force", func() {
			b := new(buildfile.Buildfile)
			n := NPM{
				Email:    "foo@bar.com",
				Username: "foo",
				Password: "bar",
				Folder:   "/path/to/repo",
				Force:    true,
			}

			n.Write(b)
			g.Assert(strings.Contains(b.String(), "npm publish /path/to/repo --force")).Equal(true)
		})

		g.It("Should set tag", func() {
			b := new(buildfile.Buildfile)
			n := NPM{
				Email:    "foo@bar.com",
				Username: "foo",
				Password: "bar",
				Folder:   "/path/to/repo",
				Tag:      "1.0.0",
			}

			n.Write(b)
			g.Assert(strings.Contains(b.String(), "npm publish /path/to/repo --tag 1.0.0")).Equal(true)
		})

		g.It("Should set registry", func() {
			b := new(buildfile.Buildfile)
			n := NPM{
				Email:    "foo@bar.com",
				Username: "foo",
				Password: "bar",
				Folder:   "/path/to/repo",
				Registry: "https://npmjs.com",
			}

			n.Write(b)
			g.Assert(strings.Contains(b.String(), "npm config set registry https://npmjs.com")).Equal(true)
		})

		g.It("Should set always-auth", func() {
			b := new(buildfile.Buildfile)
			n := NPM{
				Email:      "foo@bar.com",
				Username:   "foo",
				Password:   "bar",
				Folder:     "/path/to/repo",
				AlwaysAuth: true,
			}

			n.Write(b)
			g.Assert(strings.Contains(b.String(), CmdAlwaysAuth)).Equal(true)
		})

		g.It("Should skip when no username or password", func() {
			b := new(buildfile.Buildfile)
			n := new(NPM)

			n.Write(b)
			g.Assert(b.String()).Equal("")
		})

		g.It("Should use default username or password", func() {
			b := new(buildfile.Buildfile)
			n := new(NPM)

			expected := `sh -c "cat <<EOF > ~/.npmrc\n_auth = $(echo \"foo:bar\" | tr -d \"\\r\\n\" | base64)\nemail = foo@bar.com\nEOF"`

			var user, pass, email string = "foo", "bar", "foo@bar.com"
			DefaultUser = &user
			DefaultPass = &pass
			DefaultEmail = &email

			n.Write(b)
			g.Assert(strings.Contains(b.String(), expected)).Equal(true)
		})

		g.It("Should create npmrc", func() {
			b := new(buildfile.Buildfile)
			n := NPM{
				Email:      "foo@bar.com",
				Username:   "foo",
				Password:   "bar",
				Folder:     "/path/to/repo",
				AlwaysAuth: true,
			}

			expected := `sh -c "cat <<EOF > ~/.npmrc\n_auth = $(echo \"foo:bar\" | tr -d \"\\r\\n\" | base64)\nemail = foo@bar.com\nEOF"`

			n.Write(b)
			g.Assert(strings.Contains(b.String(), expected)).Equal(true)
		})
	})
}
