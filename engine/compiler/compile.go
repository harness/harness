package compiler

import (
	"github.com/drone/drone/engine/runner"
	"github.com/drone/drone/engine/runner/parse"

	yaml "github.com/drone/drone/engine/compiler/parse"
)

// Compiler compiles the Yaml file to the intermediate representation.
type Compiler struct {
	trans []Transform
}

func New() *Compiler {
	return &Compiler{}
}

// Transforms sets the compiler transforms use to transform the intermediate
// representation during compilation.
func (c *Compiler) Transforms(trans []Transform) *Compiler {
	c.trans = append(c.trans, trans...)
	return c
}

// CompileString compiles the Yaml configuration string and returns
// the intermediate representation for the interpreter.
func (c *Compiler) CompileString(in string) (*runner.Spec, error) {
	return c.Compile([]byte(in))
}

// CompileString compiles the Yaml configuration file and returns
// the intermediate representation for the interpreter.
func (c *Compiler) Compile(in []byte) (*runner.Spec, error) {
	root, err := yaml.Parse(in)
	if err != nil {
		return nil, err
	}
	if err := root.Walk(c.walk); err != nil {
		return nil, err
	}

	config := &runner.Spec{}
	tree := parse.NewTree()

	// pod section
	if root.Pod != nil {
		node, ok := root.Pod.(*yaml.ContainerNode)
		if ok {
			config.Containers = append(config.Containers, &node.Container)
			tree.Append(parse.NewRunNode().SetName(node.Container.Name).SetDetach(true))
		}
	}

	// cache section
	if root.Cache != nil {
		node, ok := root.Cache.(*yaml.ContainerNode)
		if ok && !node.Disabled {
			config.Containers = append(config.Containers, &node.Container)
			tree.Append(parse.NewRunNode().SetName(node.Container.Name))
		}
	}

	// clone section
	if root.Clone != nil {
		node, ok := root.Clone.(*yaml.ContainerNode)
		if ok && !node.Disabled {
			config.Containers = append(config.Containers, &node.Container)
			tree.Append(parse.NewRunNode().SetName(node.Container.Name))
		}
	}

	// services section
	for _, container := range root.Services {
		node, ok := container.(*yaml.ContainerNode)
		if !ok || node.Disabled {
			continue
		}

		config.Containers = append(config.Containers, &node.Container)
		tree.Append(parse.NewRunNode().SetName(node.Container.Name).SetDetach(true))
	}

	// pipeline section
	for i, container := range root.Script {
		node, ok := container.(*yaml.ContainerNode)
		if !ok || node.Disabled {
			continue
		}

		config.Containers = append(config.Containers, &node.Container)

		// step 1: lookahead to see if any status=failure exist
		list := parse.NewListNode()
		for ii, next := range root.Script {
			if i >= ii {
				continue
			}
			node, ok := next.(*yaml.ContainerNode)
			if !ok || node.Disabled || !node.OnFailure() {
				continue
			}

			list.Append(
				parse.NewRecoverNode().SetBody(
					parse.NewRunNode().SetName(
						node.Container.Name,
					),
				),
			)
		}
		// step 2: if yes, collect these and append to "error" node
		if len(list.Body) == 0 {
			tree.Append(parse.NewRunNode().SetName(node.Container.Name))
		} else {
			errorNode := parse.NewErrorNode()
			errorNode.SetBody(parse.NewRunNode().SetName(node.Container.Name))
			errorNode.SetDefer(list)
			tree.Append(errorNode)
		}
	}

	config.Nodes = tree
	return config, nil
}

func (c *Compiler) walk(node yaml.Node) (err error) {
	for _, trans := range c.trans {
		switch v := node.(type) {
		case *yaml.BuildNode:
			err = trans.VisitBuild(v)
		case *yaml.ContainerNode:
			err = trans.VisitContainer(v)
		case *yaml.NetworkNode:
			err = trans.VisitNetwork(v)
		case *yaml.VolumeNode:
			err = trans.VisitVolume(v)
		case *yaml.RootNode:
			err = trans.VisitRoot(v)
		}
		if err != nil {
			break
		}
	}
	return err
}
