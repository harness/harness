package builder

import "github.com/drone/drone/common"

// Builder represents a build execution tree.
type Builder struct {
	builds Node
	deploy Node
	notify Node
}

// Run runs the build, deploy and notify nodes
// in the build tree.
func (b *Builder) Run(build *B) error {
	var err error
	err = b.RunBuild(build)
	if err != nil {
		return err
	}
	err = b.RunDeploy(build)
	if err != nil {
		return err
	}
	return b.RunNotify(build)
}

// RunBuild runs only the build node.
func (b *Builder) RunBuild(build *B) error {
	return b.builds.Run(build)
}

// RunDeploy runs only the deploy node.
func (b *Builder) RunDeploy(build *B) error {
	return b.notify.Run(build)
}

// RunNotify runs on the notify node.
func (b *Builder) RunNotify(build *B) error {
	return b.notify.Run(build)
}

func (b *Builder) HasDeploy() bool {
	return len(b.deploy.(serialNode)) != 0
}

func (b *Builder) HasNotify() bool {
	return len(b.notify.(serialNode)) != 0
}

// Load loads a build configuration file.
func Load(conf *common.Config) *Builder {
	var (
		builds  []Node
		deploys []Node
		notifys []Node
	)

	for _, step := range conf.Compose {
		builds = append(builds, &serviceNode{step}) // compose
	}
	builds = append(builds, &batchNode{conf.Setup}) // setup
	if conf.Clone != nil {
		builds = append(builds, &batchNode{conf.Clone}) // clone
	}
	builds = append(builds, &batchNode{conf.Build}) // build

	for _, step := range conf.Publish {
		deploys = append(deploys, &batchNode{step}) // publish
	}
	for _, step := range conf.Deploy {
		deploys = append(deploys, &batchNode{step}) // deploy
	}
	for _, step := range conf.Notify {
		notifys = append(notifys, &batchNode{step}) // notify
	}
	return &Builder{
		serialNode(builds),
		serialNode(deploys),
		serialNode(notifys),
	}
}
