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
			g.Assert(strings.Contains(out, "npm publish /path/to/repo\n")).Equal(true)
			g.Assert(strings.Contains(out, "\nnpm set")).Equal(false)
			g.Assert(strings.Contains(out, "\nnpm config set")).Equal(false)
		})

		g.It("Should use current directory if folder is empty", func() {
			b := new(buildfile.Buildfile)
			n := NPM{
				Email:    "foo@bar.com",
				Username: "foo",
				Password: "bar",
			}

			n.Write(b)
			g.Assert(strings.Contains(b.String(), "npm publish .\n")).Equal(true)
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
			g.Assert(strings.Contains(b.String(), "\n_NPM_PACKAGE_TAG=\"1.0.0\"\n")).Equal(true)
			g.Assert(strings.Contains(b.String(), "npm tag ${_NPM_PACKAGE_NAME} ${_NPM_PACKAGE_TAG}\n")).Equal(true)
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
			g.Assert(strings.Contains(b.String(), "\nnpm config set registry https://npmjs.com\n")).Equal(true)
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

			expected := `cat <<EOF > ~/.npmrc
_auth = $(echo "foo:bar" | tr -d "\r\n" | base64)
email = foo@bar.com
EOF`

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

			expected := `cat <<EOF > ~/.npmrc
_auth = $(echo "foo:bar" | tr -d "\r\n" | base64)
email = foo@bar.com
EOF`

			n.Write(b)
			g.Assert(strings.Contains(b.String(), expected)).Equal(true)
		})
	})
}
