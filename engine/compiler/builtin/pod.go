package builtin

import (
	"fmt"

	"github.com/drone/drone/engine/compiler/parse"
	"github.com/drone/drone/engine/runner"
)

type podOp struct {
	visitor
	name string
}

// NewPodOp returns a transformer that configures an ambassador container
// providing shared networking and container volumes.
func NewPodOp(name string) Visitor {
	return &podOp{
		name: name,
	}
}

func (v *podOp) VisitContainer(node *parse.ContainerNode) error {
	if node.Container.Network == "" {
		parent := fmt.Sprintf("container:%s", v.name)
		node.Container.Network = parent
	}
	node.Container.VolumesFrom = append(node.Container.VolumesFrom, v.name)
	return nil
}

func (v *podOp) VisitRoot(node *parse.RootNode) error {
	service := node.NewServiceNode()
	service.Container = runner.Container{
		Name:       v.name,
		Alias:      "ambassador",
		Image:      "busybox",
		Entrypoint: []string{"/bin/sleep"},
		Command:    []string{"86400"},
		Volumes:    []string{node.Path, node.Base},
		// Entrypoint: []string{"/bin/sh", "-c"},
		// Volumes:    []string{node.Base},
		// Command:    []string{
		// 	fmt.Sprintf("mkdir -p %s; sleep 86400", node.Path),
		// },
	}

	node.Pod = service
	return nil
}
