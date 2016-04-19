package parse

import (
	"testing"

	"github.com/franela/goblin"
)

func TestParse(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("Parser", func() {
		g.Describe("given a yaml file", func() {

			g.It("should unmarshal a string", func() {
				out, err := ParseString(sampleYaml)
				if err != nil {
					g.Fail(err)
				}
				g.Assert(out.Image).Equal("hello-world")
				g.Assert(out.Base).Equal("/go")
				g.Assert(out.Path).Equal("src/github.com/octocat/hello-world")
				g.Assert(out.Build.(*BuildNode).Context).Equal(".")
				g.Assert(out.Build.(*BuildNode).Dockerfile).Equal("Dockerfile")
				g.Assert(out.Cache.(*ContainerNode).Vargs["mount"]).Equal("node_modules")
				g.Assert(out.Clone.(*ContainerNode).Container.Image).Equal("git")
				g.Assert(out.Clone.(*ContainerNode).Vargs["depth"]).Equal(1)
				g.Assert(out.Volumes[0].(*VolumeNode).Name).Equal("custom")
				g.Assert(out.Volumes[0].(*VolumeNode).Driver).Equal("blockbridge")
				g.Assert(out.Networks[0].(*NetworkNode).Name).Equal("custom")
				g.Assert(out.Networks[0].(*NetworkNode).Driver).Equal("overlay")
				g.Assert(out.Services[0].(*ContainerNode).Container.Name).Equal("database")
				g.Assert(out.Services[0].(*ContainerNode).Container.Image).Equal("mysql")
				g.Assert(out.Script[0].(*ContainerNode).Container.Name).Equal("test")
				g.Assert(out.Script[0].(*ContainerNode).Container.Image).Equal("golang")
				g.Assert(out.Script[0].(*ContainerNode).Commands).Equal([]string{"go install", "go test"})
				g.Assert(out.Script[0].(*ContainerNode).String()).Equal(NodeShell)
				g.Assert(out.Script[1].(*ContainerNode).Container.Name).Equal("build")
				g.Assert(out.Script[1].(*ContainerNode).Container.Image).Equal("golang")
				g.Assert(out.Script[1].(*ContainerNode).Commands).Equal([]string{"go build"})
				g.Assert(out.Script[1].(*ContainerNode).String()).Equal(NodeShell)
				g.Assert(out.Script[2].(*ContainerNode).Container.Name).Equal("notify")
				g.Assert(out.Script[2].(*ContainerNode).Container.Image).Equal("slack")
				g.Assert(out.Script[2].(*ContainerNode).String()).Equal(NodePlugin)
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

clone:
  image: git
  depth: 1

cache:
  mount: node_modules

script:
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
