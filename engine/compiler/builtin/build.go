package builtin

import (
	"github.com/drone/drone/engine/compiler/parse"
	"github.com/drone/drone/engine/runner"
)

// BuildOp is a transform operation that converts the build section of the Yaml
// to a step in the pipeline responsible for building the Docker image.
func BuildOp(node parse.Node) error {
	build, ok := node.(*parse.BuildNode)
	if !ok {
		return nil
	}
	if build.Context == "" {
		return nil
	}

	root := node.Root()
	builder := root.NewContainerNode()

	command := []string{
		"build",
		"--force-rm",
		"-f", build.Dockerfile,
		"-t", root.Image,
		build.Context,
	}

	builder.Container = runner.Container{
		Image:      "docker:apline",
		Volumes:    []string{"/var/run/docker.sock:/var/run/docker.sock"},
		Entrypoint: []string{"/usr/local/bin/docker"},
		Command:    command,
		WorkingDir: root.Path,
	}

	root.Services = append(root.Services, builder)
	return nil
}
