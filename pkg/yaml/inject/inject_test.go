package inject

import (
	"testing"

	"github.com/franela/goblin"
)

func Test_Inject(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Inject params", func() {

		g.It("Should replace vars with $$", func() {
			s := "echo $$FOO $BAR"
			m := map[string]string{}
			m["FOO"] = "BAZ"
			g.Assert("echo BAZ $BAR").Equal(Inject(s, m))
		})

		g.It("Should not replace vars with single $", func() {
			s := "echo $FOO $BAR"
			m := map[string]string{}
			m["FOO"] = "BAZ"
			g.Assert(s).Equal(Inject(s, m))
		})

		g.It("Should not replace vars in nil map", func() {
			s := "echo $$FOO $BAR"
			g.Assert(s).Equal(Inject(s, nil))
		})
	})
}

func Test_InjectSafe(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Safely Inject params", func() {

		m := map[string]string{}
		m["TOKEN"] = "FOO"
		m["SECRET"] = "BAR"
		c, _ := parse(InjectSafe(yml, m))

		g.It("Should replace vars in notify section", func() {
			g.Assert(c.Deploy["digital_ocean"].Config["token"]).Equal("FOO")
			g.Assert(c.Deploy["digital_ocean"].Config["secret"]).Equal("BAR")
		})

		g.It("Should not replace vars in script section", func() {
			g.Assert(c.Build.Config["commands"].([]interface{})[0]).Equal("echo $$TOKEN")
			g.Assert(c.Build.Config["commands"].([]interface{})[1]).Equal("echo $$SECRET")
		})
	})
}

var yml = `
build:
  image: foo
  commands:
    - echo $$TOKEN
    - echo $$SECRET
deploy:
  digital_ocean:
    token: $$TOKEN
    secret: $$SECRET
`
