package script

import (
	"github.com/franela/goblin"
	"testing"
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
