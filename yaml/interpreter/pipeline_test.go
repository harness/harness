package interpreter

import (
	"fmt"
	"testing"

	"github.com/drone/drone/yaml"
)

func TestInterpreter(t *testing.T) {

	conf, err := yaml.ParseString(sampleYaml)
	if err != nil {
		t.Fatal(err)
	}

	pipeline := Load(conf, nil)
	pipeline.pipe <- &Line{Out: "foo"}
	pipeline.pipe <- &Line{Out: "bar"}
	pipeline.pipe <- &Line{Out: "baz"}
	for {
		select {
		case <-pipeline.Done():
			fmt.Println("GOT DONE")
			return

		case line := <-pipeline.Pipe():
			fmt.Println(line.String())

		case <-pipeline.Next():
			pipeline.Exec()
		}
	}
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
