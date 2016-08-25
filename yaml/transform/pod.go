package transform

import (
	"encoding/base64"
	"fmt"

	"github.com/drone/drone/yaml"

	"github.com/gorilla/securecookie"
)

type ambassador struct {
	image      string
	entrypoint []string
	command    []string
}

// default linux amd64 ambassador
var defaultAmbassador = ambassador{
	image:      "busybox:latest",
	entrypoint: []string{"/bin/sleep"},
	command:    []string{"86400"},
}

// lookup ambassador configuration by architecture and os
var lookupAmbassador = map[string]ambassador{
	"linux/amd64": {
		image:      "busybox:latest",
		entrypoint: []string{"/bin/sleep"},
		command:    []string{"86400"},
	},
	"linux/arm": {
		image:      "armhf/alpine:latest",
		entrypoint: []string{"/bin/sleep"},
		command:    []string{"86400"},
	},
}

// Pod transforms the containers in the Yaml to use Pod networking, where every
// container shares the localhost connection.
func Pod(c *yaml.Config, platform string) error {

	rand := base64.RawURLEncoding.EncodeToString(
		securecookie.GenerateRandomKey(8),
	)

	// choose the ambassador configuration based on os and architecture
	conf, ok := lookupAmbassador[platform]
	if !ok {
		conf = defaultAmbassador
	}

	ambassador := &yaml.Container{
		ID:          fmt.Sprintf("drone_ambassador_%s", rand),
		Name:        "ambassador",
		Image:       conf.image,
		Detached:    true,
		Entrypoint:  conf.entrypoint,
		Command:     conf.command,
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
