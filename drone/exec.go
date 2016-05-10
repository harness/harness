package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/drone/drone/build"
	"github.com/drone/drone/model"
	"github.com/drone/drone/yaml"
	"github.com/drone/drone/yaml/expander"
	"github.com/drone/drone/yaml/transform"

	"github.com/codegangsta/cli"
	"github.com/samalba/dockerclient"
)

var execCmd = cli.Command{
	Name:  "exec",
	Usage: "execute a local build",
	Action: func(c *cli.Context) {
		if err := exec(c); err != nil {
			log.Fatalln(err)
		}
	},
	Flags: []cli.Flag{
		cli.BoolTFlag{
			Name:   "local",
			Usage:  "build from local directory",
			EnvVar: "DRONE_LOCAL",
		},
		cli.StringSliceFlag{
			Name:   "plugin",
			Usage:  "plugin steps to enable",
			EnvVar: "DRONE_PLUGIN_ENABLE",
		},
		cli.StringSliceFlag{
			Name:   "secret",
			Usage:  "build secrets in KEY=VALUE format",
			EnvVar: "DRONE_SECRET",
		},
		cli.StringSliceFlag{
			Name:   "matrix",
			Usage:  "build matrix in KEY=VALUE format",
			EnvVar: "DRONE_MATRIX",
		},
		cli.DurationFlag{
			Name:   "timeout",
			Usage:  "build timeout for inactivity",
			Value:  time.Hour,
			EnvVar: "DRONE_TIMEOUT",
		},
		cli.DurationFlag{
			Name:   "duration",
			Usage:  "build duration",
			Value:  time.Hour,
			EnvVar: "DRONE_DURATION",
		},
		cli.BoolFlag{
			EnvVar: "DRONE_PLUGIN_PULL",
			Name:   "pull",
			Usage:  "always pull latest plugin images",
		},
		cli.StringFlag{
			EnvVar: "DRONE_PLUGIN_NAMESPACE",
			Name:   "namespace",
			Value:  "plugins",
			Usage:  "default plugin image namespace",
		},
		cli.StringSliceFlag{
			EnvVar: "DRONE_PLUGIN_PRIVILEGED",
			Name:   "privileged",
			Usage:  "plugins that require privileged mode",
			Value: &cli.StringSlice{
				"plugins/docker",
				"plugins/docker:*",
				"plguins/gcr",
				"plguins/gcr:*",
				"plugins/ecr",
				"plugins/ecr:*",
			},
		},

		// Docker daemon flags

		cli.StringFlag{
			EnvVar: "DOCKER_HOST",
			Name:   "docker-host",
			Usage:  "docker deamon address",
			Value:  "unix:///var/run/docker.sock",
		},
		cli.BoolFlag{
			EnvVar: "DOCKER_TLS_VERIFY",
			Name:   "docker-tls-verify",
			Usage:  "docker daemon supports tlsverify",
		},
		cli.StringFlag{
			EnvVar: "DOCKER_CERT_PATH",
			Name:   "docker-cert-path",
			Usage:  "docker certificate directory",
			Value:  "",
		},

		//
		// Please note the below flags are mirrored in the plugin starter kit and
		// should be kept synchronized.
		// https://github.com/drone/drone-plugin-starter
		//

		cli.StringFlag{
			Name:   "repo.fullname",
			Usage:  "repository full name",
			EnvVar: "DRONE_REPO",
		},
		cli.StringFlag{
			Name:   "repo.owner",
			Usage:  "repository owner",
			EnvVar: "DRONE_REPO_OWNER",
		},
		cli.StringFlag{
			Name:   "repo.name",
			Usage:  "repository name",
			EnvVar: "DRONE_REPO_NAME",
		},
		cli.StringFlag{
			Name:   "repo.type",
			Value:  "git",
			Usage:  "repository type",
			EnvVar: "DRONE_REPO_SCM",
		},
		cli.StringFlag{
			Name:   "repo.link",
			Usage:  "repository link",
			EnvVar: "DRONE_REPO_LINK",
		},
		cli.StringFlag{
			Name:   "repo.avatar",
			Usage:  "repository avatar",
			EnvVar: "DRONE_REPO_AVATAR",
		},
		cli.StringFlag{
			Name:   "repo.branch",
			Usage:  "repository default branch",
			EnvVar: "DRONE_REPO_BRANCH",
		},
		cli.BoolFlag{
			Name:   "repo.private",
			Usage:  "repository is private",
			EnvVar: "DRONE_REPO_PRIVATE",
		},
		cli.BoolFlag{
			Name:   "repo.trusted",
			Usage:  "repository is trusted",
			EnvVar: "DRONE_REPO_TRUSTED",
		},
		cli.StringFlag{
			Name:   "remote.url",
			Usage:  "git remote url",
			EnvVar: "DRONE_REMOTE_URL",
		},
		cli.StringFlag{
			Name:   "commit.sha",
			Usage:  "git commit sha",
			EnvVar: "DRONE_COMMIT_SHA",
		},
		cli.StringFlag{
			Name:   "commit.ref",
			Value:  "refs/heads/master",
			Usage:  "git commit ref",
			EnvVar: "DRONE_COMMIT_REF",
		},
		cli.StringFlag{
			Name:   "commit.branch",
			Value:  "master",
			Usage:  "git commit branch",
			EnvVar: "DRONE_COMMIT_BRANCH",
		},
		cli.StringFlag{
			Name:   "commit.message",
			Usage:  "git commit message",
			EnvVar: "DRONE_COMMIT_MESSAGE",
		},
		cli.StringFlag{
			Name:   "commit.link",
			Usage:  "git commit link",
			EnvVar: "DRONE_COMMIT_LINK",
		},
		cli.StringFlag{
			Name:   "commit.author.name",
			Usage:  "git author name",
			EnvVar: "DRONE_COMMIT_AUTHOR",
		},
		cli.StringFlag{
			Name:   "commit.author.email",
			Usage:  "git author email",
			EnvVar: "DRONE_COMMIT_AUTHOR_EMAIL",
		},
		cli.StringFlag{
			Name:   "commit.author.avatar",
			Usage:  "git author avatar",
			EnvVar: "DRONE_COMMIT_AUTHOR_AVATAR",
		},
		cli.StringFlag{
			Name:   "build.event",
			Value:  "push",
			Usage:  "build event",
			EnvVar: "DRONE_BUILD_EVENT",
		},
		cli.IntFlag{
			Name:   "build.number",
			Usage:  "build number",
			EnvVar: "DRONE_BUILD_NUMBER",
		},
		cli.IntFlag{
			Name:   "build.created",
			Usage:  "build created",
			EnvVar: "DRONE_BUILD_CREATED",
		},
		cli.IntFlag{
			Name:   "build.started",
			Usage:  "build started",
			EnvVar: "DRONE_BUILD_STARTED",
		},
		cli.IntFlag{
			Name:   "build.finished",
			Usage:  "build finished",
			EnvVar: "DRONE_BUILD_FINISHED",
		},
		cli.StringFlag{
			Name:   "build.status",
			Usage:  "build status",
			Value:  "success",
			EnvVar: "DRONE_BUILD_STATUS",
		},
		cli.StringFlag{
			Name:   "build.link",
			Usage:  "build link",
			EnvVar: "DRONE_BUILD_LINK",
		},
		cli.StringFlag{
			Name:   "build.deploy",
			Usage:  "build deployment target",
			EnvVar: "DRONE_DEPLOY_TO",
		},
		cli.BoolFlag{
			Name:   "yaml.verified",
			Usage:  "build yaml is verified",
			EnvVar: "DRONE_YAML_VERIFIED",
		},
		cli.BoolFlag{
			Name:   "yaml.signed",
			Usage:  "build yaml is signed",
			EnvVar: "DRONE_YAML_SIGNED",
		},
		cli.IntFlag{
			Name:   "prev.build.number",
			Usage:  "previous build number",
			EnvVar: "DRONE_PREV_BUILD_NUMBER",
		},
		cli.StringFlag{
			Name:   "prev.build.status",
			Usage:  "previous build status",
			EnvVar: "DRONE_PREV_BUILD_STATUS",
		},
		cli.StringFlag{
			Name:   "prev.commit.sha",
			Usage:  "previous build sha",
			EnvVar: "DRONE_PREV_COMMIT_SHA",
		},

		cli.StringFlag{
			Name:   "netrc.username",
			Usage:  "previous build sha",
			EnvVar: "DRONE_NETRC_USERNAME",
		},
		cli.StringFlag{
			Name:   "netrc.password",
			Usage:  "previous build sha",
			EnvVar: "DRONE_NETRC_PASSWORD",
		},
		cli.StringFlag{
			Name:   "netrc.machine",
			Usage:  "previous build sha",
			EnvVar: "DRONE_NETRC_MACHINE",
		},
	},
}

func exec(c *cli.Context) error {

	// get environment variables from flags
	var envs = map[string]string{}
	for _, flag := range c.Command.Flags {
		switch f := flag.(type) {
		case cli.StringFlag:
			envs[f.EnvVar] = c.String(f.Name)
		case cli.IntFlag:
			envs[f.EnvVar] = c.String(f.Name)
		case cli.BoolFlag:
			envs[f.EnvVar] = c.String(f.Name)
		}
	}

	// get matrix variales from flags
	for _, s := range c.StringSlice("matrix") {
		parts := strings.SplitN(s, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := parts[0]
		v := parts[1]
		envs[k] = v
	}

	// get secret variales from flags
	for _, s := range c.StringSlice("secret") {
		parts := strings.SplitN(s, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := parts[0]
		v := parts[1]
		envs[k] = v
	}

	// 	builtin.NewFilterOp(
	// 		c.String("prev.build.status"),
	// 		c.String("commit.branch"),
	// 		c.String("build.event"),
	// 		c.String("build.deploy"),
	// 		envs,
	// 	),
	// }

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, os.Interrupt)

	path := c.Args().First()
	if path == "" {
		path = ".drone.yml"
	}
	path, _ = filepath.Abs(path)
	dir := filepath.Dir(path)

	file, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	// unmarshal the Yaml file with expanded environment variables.
	conf, err := yaml.Parse(expander.Expand(file, envs))
	if err != nil {
		return err
	}

	tls, err := dockerclient.TLSConfigFromCertPath(c.String("docker-cert-path"))
	if err == nil {
		tls.InsecureSkipVerify = c.Bool("docker-tls-verify")
	}
	client, err := dockerclient.NewDockerClient(c.String("docker-host"), tls)
	if err != nil {
		return err
	}

	src := "src"
	if url, _ := url.Parse(c.String("repo.link")); url != nil {
		src = filepath.Join(src, url.Host, url.Path)
	}

	transform.Clone(conf, "git")
	transform.Environ(conf, envs)
	transform.DefaultFilter(conf)

	transform.PluginDisable(conf, c.StringSlice("plugin"))

	// transform.Secret(conf, secrets)
	transform.Identifier(conf)
	transform.WorkspaceTransform(conf, "/drone", src)

	if err := transform.Check(conf, c.Bool("repo.trusted")); err != nil {
		return err
	}

	transform.CommandTransform(conf)
	transform.ImagePull(conf, c.Bool("pull"))
	transform.ImageTag(conf)
	transform.ImageName(conf)
	transform.ImageNamespace(conf, c.String("namespace"))
	transform.ImageEscalate(conf, c.StringSlice("privileged"))

	if c.BoolT("local") {
		transform.ImageVolume(conf, []string{dir + ":" + conf.Workspace.Path})
	}
	transform.PluginParams(conf)
	transform.Pod(conf)

	timeout := time.After(c.Duration("duration"))

	// load the Yaml into the pipeline
	pipeline := build.Load(conf, client)
	defer pipeline.Teardown()

	// setup the build environment
	err = pipeline.Setup()
	if err != nil {
		return err
	}

	for {
		select {
		case <-pipeline.Done():
			return pipeline.Err()
		case <-sigterm:
			pipeline.Stop()
			return fmt.Errorf("interrupt received, build cancelled")
		case <-timeout:
			pipeline.Stop()
			return fmt.Errorf("maximum time limit exceeded, build cancelled")
		case <-time.After(c.Duration("timeout")):
			pipeline.Stop()
			return fmt.Errorf("terminal inactive for %v, build cancelled", c.Duration("timeout"))
		case <-pipeline.Next():

			// TODO(bradrydzewski) this entire block of code should probably get
			// encapsulated in the pipeline.
			status := model.StatusSuccess
			if pipeline.Err() != nil {
				status = model.StatusFailure
			}

			if !pipeline.Head().Constraints.Match(
				"linux/amd64",
				c.String("build.deploy"),
				c.String("build.event"),
				c.String("commit.branch"),
				status, envs) {

				pipeline.Skip()
			} else {
				pipeline.Exec()
				pipeline.Head().Environment["DRONE_STATUS"] = status
			}
		case line := <-pipeline.Pipe():
			println(line.String())
		}
	}
}
