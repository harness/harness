package build

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
