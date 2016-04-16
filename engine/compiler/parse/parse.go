package parse

import (
	"gopkg.in/yaml.v2"
)

// Parse parses a Yaml file and returns a Tree structure.
func Parse(in []byte) (*RootNode, error) {
	out := root{}
	err := yaml.Unmarshal(in, &out)
	if err != nil {
		return nil, err
	}

	root := NewRootNode()
	root.Platform = out.Platform
	root.Path = out.Workspace.Path
	root.Base = out.Workspace.Base
	root.Image = out.Image

	// append volume nodes to tree
	for _, v := range out.Volumes.volumes {
		vv := root.NewVolumeNode(v.Name)
		vv.Driver = v.Driver
		vv.DriverOpts = v.DriverOpts
		root.Volumes = append(root.Volumes, vv)
	}

	// append network nodes to tree
	for _, n := range out.Networks.networks {
		nn := root.NewNetworkNode(n.Name)
		nn.Driver = n.Driver
		nn.DriverOpts = n.DriverOpts
		root.Networks = append(root.Networks, nn)
	}

	// add the build section
	if out.Build.Context != "" {
		root.Build = &BuildNode{
			NodeType:   NodeBuild,
			Context:    out.Build.Context,
			Dockerfile: out.Build.Dockerfile,
			Args:       out.Build.Args,
			root:       root,
		}
	}

	// add the cache section
	{
		cc := root.NewCacheNode()
		cc.Container = out.Cache.ToContainer()
		cc.Conditions = out.Cache.ToConditions()
		cc.Container.Name = "cache"
		cc.Vargs = out.Cache.Vargs
		root.Cache = cc
	}

	// add the clone section
	{
		cc := root.NewCloneNode()
		cc.Conditions = out.Clone.ToConditions()
		cc.Container = out.Clone.ToContainer()
		cc.Container.Name = "clone"
		cc.Vargs = out.Clone.Vargs
		root.Clone = cc
	}

	// append services
	for _, c := range out.Services.containers {
		if c.Build != "" {
			continue
		}
		cc := root.NewServiceNode()
		cc.Conditions = c.ToConditions()
		cc.Container = c.ToContainer()
		root.Services = append(root.Services, cc)
	}

	// append scripts
	for _, c := range out.Script.containers {
		var cc *ContainerNode
		if len(c.Commands.parts) == 0 {
			cc = root.NewPluginNode()
		} else {
			cc = root.NewShellNode()
		}
		cc.Commands = c.Commands.parts
		cc.Vargs = c.Vargs
		cc.Container = c.ToContainer()
		cc.Conditions = c.ToConditions()
		root.Script = append(root.Script, cc)
	}

	return root, nil
}

// ParseString parses a Yaml string and returns a Tree structure.
func ParseString(in string) (*RootNode, error) {
	return Parse([]byte(in))
}
