package engine

import (
	"testing"

	. "github.com/franela/goblin"
)

func TestAgent(t *testing.T) {
	g := Goblin(t)
	g.Describe("Engine agent", func() {
		testdata := []struct {
			name       string
			dsn        string
			image      string
			entrypoint []string
			cmd        []string
			exp        *Agent
			err        bool
		}{
			{
				"Should parse an agent DSN",
				"image: drone/drone-exec, entrypoint: [/bin/drone-exec], cmd: [--clone, --cache, --build, --publish, --deploy]",
				"",
				[]string{},
				[]string{},
				&Agent{
					Image:      "drone/drone-exec",
					Entrypoint: []string{"/bin/drone-exec"},
					Cmd:        []string{"--clone", "--cache", "--build", "--publish", "--deploy"},
				},
				false,
			},
			{
				"Should use defaults",
				"",
				"drone/drone-exec",
				[]string{"/bin/drone-exec"},
				[]string{"--clone", "--cache", "--build", "--publish", "--deploy"},
				&Agent{
					Image:      "drone/drone-exec",
					Entrypoint: []string{"/bin/drone-exec"},
					Cmd:        []string{"--clone", "--cache", "--build", "--publish", "--deploy"},
				},
				false,
			},
			{
				"Should use partial defaults (image)",
				"image: drone/drone-exec",
				"foobar/drone-exec",
				[]string{"/bin/drone-exec"},
				[]string{"--clone", "--cache", "--build", "--publish", "--deploy"},
				&Agent{
					Image:      "drone/drone-exec",
					Entrypoint: []string{"/bin/drone-exec"},
					Cmd:        []string{"--clone", "--cache", "--build", "--publish", "--deploy"},
				},
				false,
			},
			{
				"Should use partial defaults (entrypoint)",
				"entrypoint: [/bin/drone-exec, --debug]",
				"drone/drone-exec",
				[]string{"/bin/drone-exec"},
				[]string{"--clone", "--cache", "--build", "--publish", "--deploy"},
				&Agent{
					Image:      "drone/drone-exec",
					Entrypoint: []string{"/bin/drone-exec", "--debug"},
					Cmd:        []string{"--clone", "--cache", "--build", "--publish", "--deploy"},
				},
				false,
			},
			{
				"Should use partial defaults (cmd)",
				"cmd: [--debug]",
				"drone/drone-exec",
				[]string{"/bin/drone-exec"},
				[]string{"--clone", "--cache", "--build", "--publish", "--deploy"},
				&Agent{
					Image:      "drone/drone-exec",
					Entrypoint: []string{"/bin/drone-exec"},
					Cmd:        []string{"--debug"},
				},
				false,
			},
			{
				"Should allow environment configuration",
				"env: [FOO=bar, THIS=that]",
				"drone/drone-exec",
				[]string{"/bin/drone-exec"},
				[]string{"--clone", "--cache", "--build", "--publish", "--deploy"},
				&Agent{
					Image:      "drone/drone-exec",
					Entrypoint: []string{"/bin/drone-exec"},
					Cmd:        []string{"--clone", "--cache", "--build", "--publish", "--deploy"},
					Env:        []string{"FOO=bar", "THIS=that"},
				},
				false,
			},
		}
		for i, _ := range testdata {
			data := testdata[i]
			g.It(data.name, func() {
				a, err := NewAgent(data.dsn, data.image, data.entrypoint, data.cmd)
				if data.err == false {
					g.Assert(err).Equal(nil)
					g.Assert(a).Equal(data.exp)
				} else {
					g.Assert(err != nil).IsFalse("Expected an error, got nil")
				}
			})
		}
	})
}
