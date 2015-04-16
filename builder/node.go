package builder

import (
	"sync"

	"github.com/drone/drone/common"
)

// Node is an element in the build execution tree.
type Node interface {
	Run(*B) error
}

// parallelNode runs a set of build nodes in parallel.
type parallelNode []Node

func (n parallelNode) Run(b *B) error {
	var wg sync.WaitGroup
	for _, node := range n {
		wg.Add(1)

		go func(node Node) {
			defer wg.Done()
			node.Run(b)
		}(node)
	}
	wg.Wait()
	return nil
}

// serialNode runs a set of build nodes in sequential order.
type serialNode []Node

func (n serialNode) Run(b *B) error {
	for _, node := range n {
		err := node.Run(b)
		if err != nil {
			return err
		}
		if b.ExitCode() != 0 {
			return nil
		}
	}
	return nil
}

// batchNode runs a container and blocks until complete.
type batchNode struct {
	step *common.Step
}

func (n *batchNode) Run(b *B) error {

	// switch {
	// case n.step.Condition == nil:
	// case n.step.Condition.MatchBranch(b.Commit.Branch) == false:
	// 	return nil
	// case n.step.Condition.MatchOwner(b.Repo.Owner) == false:
	// 	return nil
	// }

	// creates the container conf
	conf := toContainerConfig(n.step)
	if n.step.Config != nil {
		conf.Cmd = toCommand(b, n.step)
	}

	// inject environment vars
	injectEnv(b, conf)

	name, err := b.Run(conf)
	if err != nil {
		return err
	}

	// streams the logs to the build results
	rc, err := b.Logs(name)
	if err != nil {
		return err
	}
	StdCopy(b, b, rc)
	//io.Copy(b, rc)

	// inspects the results and writes the
	// build result exit code
	info, err := b.Inspect(name)
	if err != nil {
		return err
	}
	b.Exit(info.State.ExitCode)
	return nil
}

// serviceNode runs a container, blocking, writes output, uses config section
type serviceNode struct {
	step *common.Step
}

func (n *serviceNode) Run(b *B) error {
	conf := toContainerConfig(n.step)
	_, err := b.Run(conf)
	return err
}
