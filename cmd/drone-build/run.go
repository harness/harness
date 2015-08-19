package main

import (
	"path/filepath"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/samalba/dockerclient"
	common "github.com/drone/drone/pkg/types"
	"github.com/drone/drone/pkg/yaml"
	"github.com/drone/drone/pkg/yaml/inject"
)

type Context struct {
	// Links  *common.Link
	Clone   *common.Clone   `json:"clone"`
	Repo    *common.Repo    `json:"repo"`
	Build   *common.Build   `json:"build"`
	Job     *common.Job     `json:"job"`
	Keys    *common.Keypair `json:"keys"`
	Netrc   *common.Netrc   `json:"netrc"`
	Yaml    []byte          `json:"yaml"`
	Env     []string        `json:"environment"`
	Plugins []string        `json:"plugins"`

	Conf   *common.Config `json:"-"`
	infos  []*dockerclient.ContainerInfo
	client dockerclient.Client
}

func setup(c *Context) error {
	var err error
	var opts = parser.Opts{
		Network:    false,
		Privileged: false,
		Volumes:    false,
		Whitelist:  c.Plugins,
	}

	// if repository is trusted the build may specify
	// custom volumes, networking and run in trusted mode.
	if c.Repo.Trusted {
		opts.Network = true
		opts.Privileged = true
		opts.Volumes = true
	}

	// inject the matrix parameters into the yaml
	injected := inject.Inject(string(c.Yaml), c.Job.Environment)
	c.Conf, err = parser.ParseSingle(injected, &opts, c.Repo)
	if err != nil {
		return err
	}

	// and append the matrix parameters as environment
	// variables for the build
	for k, v := range c.Job.Environment {
		env := k + "=" + v
		c.Conf.Build.Environment = append(c.Conf.Build.Environment, env)
	}

	// and append drone, jenkins, travis and other
	// environment variables that may be of use.
	for k, v := range toEnv(c) {
		env := k + "=" + v
		c.Conf.Build.Environment = append(c.Conf.Build.Environment, env)
	}

	return nil
}

type execFunc func(c *Context) (int, error)

func execClone(c *Context) (int, error) {
	conf := toContainerConfig(c.Conf.Clone)
	conf.Cmd = toCommand(c, c.Conf.Clone)
	info, err := run(c.client, conf, c.Conf.Clone.Pull)
	if err != nil {
		return 255, err
	}
	return info.State.ExitCode, nil
}

func execBuild(c *Context) (int, error) {
	conf := toContainerConfig(c.Conf.Build)
	conf.Entrypoint = []string{"/bin/sh", "-e"}
	conf.Cmd = []string{"/drone/bin/build.sh"}
	info, err := run(c.client, conf, c.Conf.Build.Pull)
	if err != nil {
		return 255, err
	}
	return info.State.ExitCode, nil
}

func execSetup(c *Context) (int, error) {
	conf := toContainerConfig(c.Conf.Setup)
	conf.Cmd = toCommand(c, c.Conf.Setup)
	info, err := run(c.client, conf, c.Conf.Setup.Pull)
	if err != nil {
		return 255, err
	}
	return info.State.ExitCode, nil
}

func execDeploy(c *Context) (int, error) {
	return runSteps(c, c.Conf.Deploy)
}

func execPublish(c *Context) (int, error) {
	return runSteps(c, c.Conf.Publish)
}

func execNotify(c *Context) (int, error) {
	return runSteps(c, c.Conf.Notify)
}

func execCompose(c *Context) (int, error) {
	for _, step := range c.Conf.Compose {
		conf := toContainerConfig(step)
		_, err := daemon(c.client, conf, step.Pull)
		if err != nil {
			return 0, err
		}
	}
	return 0, nil
}

func runSteps(c *Context, steps map[string]*common.Step) (int, error) {
	for _, step := range steps {

		// verify the step matches the branch
		// and other specifications
		if step.Condition != nil {
			if !step.Condition.MatchOwner(c.Repo.Owner) ||
				!step.Condition.MatchBranch(c.Clone.Branch) ||
				!step.Condition.MatchMatrix(c.Job.Environment) {
				continue
			}
		}

		conf := toContainerConfig(step)
		conf.Cmd = toCommand(c, step)

		// append global environment variables
		conf.Env = append(conf.Env, c.Env...)

		info, err := run(c.client, conf, step.Pull)
		if err != nil {
			return 255, err
		} else if info.State.ExitCode != 0 {
			return info.State.ExitCode, nil
		}
	}
	return 0, nil
}
