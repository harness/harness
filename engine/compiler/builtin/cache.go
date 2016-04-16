package builtin

import (
	"github.com/drone/drone/engine/runner"
	"github.com/drone/drone/engine/compiler/parse"
)

type cacheOp struct {
	visitor
	enable bool
	plugin string
	mount  string
}

// NewCacheOp returns a transformer that configures the default cache plugin.
func NewCacheOp(plugin, mount string, enable bool) Visitor {
	return &cacheOp{
		mount: mount,
		enable: enable,
		plugin: plugin,
	}
}

func (v *cacheOp) VisitContainer(node *parse.ContainerNode) error {
	if node.Type() != parse.NodeCache {
		return nil
	}
	if len(node.Vargs) == 0 || v.enable == false {
		node.Disabled = true
		return nil
	}

	if node.Container.Name == "" {
		node.Container.Name = "cache"
	}
	if node.Container.Image == "" {
		node.Container.Image = v.plugin
	}

	// discard any other cache properties except the image name.
	// everything else is discard for security reasons.
	node.Container = runner.Container{
		Name: node.Container.Name,
		Alias: node.Container.Alias,
		Image: node.Container.Image,
		Volumes: []string{
			v.mount + ":/cache",
		},
	}

	// this is a hack until I can come up with a better solution.
	// this copies the clone name, and appends at the end of the
	// build. When it is executed a second time the build should
	// have a completed status, so it knows to cache instead
	// of restore.
	cache := node.Root().NewCacheNode()
	cache.Vargs = node.Vargs
	cache.Container = node.Container
	node.Root().Script = append(node.Root().Script, cache)
	return nil
}