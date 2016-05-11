package transform

import (
	"encoding/base64"
	"fmt"

	"github.com/drone/drone/yaml"

	"github.com/gorilla/securecookie"
)

// Pod transforms the containers in the Yaml to use Pod networking, where every
// container shares the localhost connection.
func Pod(c *yaml.Config) error {

	rand := base64.RawURLEncoding.EncodeToString(
		securecookie.GenerateRandomKey(8),
	)

	ambassador := &yaml.Container{
		ID:          fmt.Sprintf("drone_ambassador_%s", rand),
		Name:        "ambassador",
		Image:       "busybox:latest",
		Detached:    true,
		Entrypoint:  []string{"/bin/sleep"},
		Command:     []string{"86400"},
		Volumes:     []string{c.Workspace.Path, c.Workspace.Base},
		Environment: map[string]string{},
	}
	network := fmt.Sprintf("container:%s", ambassador.ID)

	var containers []*yaml.Container
	containers = append(containers, c.Pipeline...)
	containers = append(containers, c.Services...)

	for _, container := range containers {
		container.VolumesFrom = append(container.VolumesFrom, ambassador.ID)
		if container.Network == "" {
			container.Network = network
		}
	}

	c.Services = append([]*yaml.Container{ambassador}, c.Services...)
	return nil
}

// func (v *podOp) VisitContainer(node *parse.ContainerNode) error {
// 	if node.Container.Network == "" {
// 		parent := fmt.Sprintf("container:%s", v.name)
// 		node.Container.Network = parent
// 	}
// 	node.Container.VolumesFrom = append(node.Container.VolumesFrom, v.name)
// 	return nil
// }
//
// func (v *podOp) VisitRoot(node *parse.RootNode) error {
//
//
// 	node.Pod = service
// 	return nil
// }
