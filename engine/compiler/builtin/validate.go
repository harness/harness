package builtin

import (
	"fmt"
	"path/filepath"

	"github.com/drone/drone/engine/compiler/parse"
)

type validateOp struct {
	visitor
	plugins []string
	trusted bool
}

// NewValidateOp returns a linter that checks container configuration.
func NewValidateOp(trusted bool, plugins []string) Visitor {
	return &validateOp{
		trusted: trusted,
		plugins: plugins,
	}
}

func (v *validateOp) VisitContainer(node *parse.ContainerNode) error {
	switch node.NodeType {
	case parse.NodePlugin, parse.NodeCache, parse.NodeClone:
		if err := v.validatePlugins(node); err != nil {
			return err
		}
	}
	if node.NodeType == parse.NodePlugin {
		if err := v.validatePluginConfig(node); err != nil {
			return err
		}
	}
	return v.validateConfig(node)
}

// validate the plugin image and return an error if the plugin
// image does not match the whitelist.
func (v *validateOp) validatePlugins(node *parse.ContainerNode) error {
	match := false
	for _, pattern := range v.plugins {
		ok, err := filepath.Match(pattern, node.Container.Image)
		if ok && err == nil {
			match = true
			break
		}
	}
	if !match {
		return fmt.Errorf(
			"Plugin %s is not in the whitelist",
			node.Container.Image,
		)
	}
	return nil
}

// validate the plugin command and entrypoint and return an error
// the user attempts to set or override these values.
func (v *validateOp) validatePluginConfig(node *parse.ContainerNode) error {
	if len(node.Container.Entrypoint) != 0 {
		return fmt.Errorf("Cannot set plugin Entrypoint")
	}
	if len(node.Container.Command) != 0 {
		return fmt.Errorf("Cannot set plugin Command")
	}
	return nil
}

// validate the container configuration and return an error if
// restricted configurations are used.
func (v *validateOp) validateConfig(node *parse.ContainerNode) error {
	if v.trusted {
		return nil
	}
	if node.Container.Privileged {
		return fmt.Errorf("Insufficient privileges to use privileged mode")
	}
	if len(node.Container.DNS) != 0 {
		return fmt.Errorf("Insufficient privileges to use custom dns")
	}
	if len(node.Container.DNSSearch) != 0 {
		return fmt.Errorf("Insufficient privileges to use dns_search")
	}
	if len(node.Container.Devices) != 0 {
		return fmt.Errorf("Insufficient privileges to use devices")
	}
	if len(node.Container.ExtraHosts) != 0 {
		return fmt.Errorf("Insufficient privileges to use extra_hosts")
	}
	if len(node.Container.Network) != 0 {
		return fmt.Errorf("Insufficient privileges to override the network")
	}
	if node.Container.OomKillDisable {
		return fmt.Errorf("Insufficient privileges to disable oom_kill")
	}
	if len(node.Container.Volumes) != 0 && node.Type() != parse.NodeCache {
		return fmt.Errorf("Insufficient privileges to use volumes")
	}
	if len(node.Container.VolumesFrom) != 0 {
		return fmt.Errorf("Insufficient privileges to use volumes_from")
	}
	return nil
}

// validate the environment configuration and return an error if
// an attempt is made to override system environment variables.
// func (v *validateOp) validateEnvironment(node *parse.ContainerNode) error {
// 	for key := range node.Container.Environment {
// 		upper := strings.ToUpper(key)
// 		switch {
// 		case strings.HasPrefix(upper, "DRONE_"):
// 			return fmt.Errorf("Cannot set or override DRONE_ environment variables")
// 		case strings.HasPrefix(upper, "PLUGIN_"):
// 			return fmt.Errorf("Cannot set or override PLUGIN_ environment variables")
// 		}
// 	}
// 	return nil
// }
