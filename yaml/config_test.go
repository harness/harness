package yaml

import (
	"testing"

	"github.com/franela/goblin"
)

func TestParse(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("Parser", func() {
		g.Describe("Given a yaml file", func() {

			g.It("Should unmarshal a string", func() {
				out, err := ParseString(sampleYaml)
				if err != nil {
					g.Fail(err)
				}
				g.Assert(out.Image).Equal("hello-world")
				g.Assert(out.Workspace.Base).Equal("/go")
				g.Assert(out.Workspace.Path).Equal("src/github.com/octocat/hello-world")
				g.Assert(out.Build.Context).Equal(".")
				g.Assert(out.Build.Dockerfile).Equal("Dockerfile")
				g.Assert(out.Volumes[0].Name).Equal("custom")
				g.Assert(out.Volumes[0].Driver).Equal("blockbridge")
				g.Assert(out.Networks[0].Name).Equal("custom")
				g.Assert(out.Networks[0].Driver).Equal("overlay")
				g.Assert(out.Services[0].Name).Equal("database")
				g.Assert(out.Services[0].Image).Equal("mysql")
				g.Assert(out.Pipeline[0].Name).Equal("test")
				g.Assert(out.Pipeline[0].Image).Equal("golang")
				g.Assert(out.Pipeline[0].Commands).Equal([]string{"go install", "go test"})
				g.Assert(out.Pipeline[1].Name).Equal("build")
				g.Assert(out.Pipeline[1].Image).Equal("golang")
				g.Assert(out.Pipeline[1].Commands).Equal([]string{"go build"})
				g.Assert(out.Pipeline[2].Name).Equal("notify")
				g.Assert(out.Pipeline[2].Image).Equal("slack")
			})
			// Check to make sure variable expansion works in yaml.MapSlice
			g.It("Should unmarshal variables", func() {
				out, err := ParseString(sampleVarYaml)
				if err != nil {
					g.Fail(err)
				}
				g.Assert(out.Pipeline[0].Name).Equal("notify_fail")
				g.Assert(out.Pipeline[0].Image).Equal("plugins/slack")
				g.Assert(len(out.Pipeline[0].Constraints.Event.Include)).Equal(0)
				g.Assert(out.Pipeline[1].Name).Equal("notify_success")
				g.Assert(out.Pipeline[1].Image).Equal("plugins/slack")
				g.Assert(out.Pipeline[1].Constraints.Event.Include).Equal([]string{"success"})
			})
		})
	})
}

var sampleYaml = `
image: hello-world
build:
  context: .
  dockerfile: Dockerfile

workspace:
  path: src/github.com/octocat/hello-world
  base: /go

pipeline:
  test:
    image: golang
    commands:
      - go install
      - go test
  build:
    image: golang
    commands:
      - go build
    when:
      event: push
  notify:
    image: slack
    channel: dev
    when:
      event: failure

services:
  database:
    image: mysql

networks:
  custom:
    driver: overlay

volumes:
  custom:
    driver: blockbridge
`

var sampleVarYaml = `
_slack: &SLACK
  image: plugins/slack

pipeline:
  notify_fail: *SLACK
  notify_success:
    << : *SLACK
    when:
      event: success
`
