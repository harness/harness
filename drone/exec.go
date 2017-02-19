package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/drone/drone/agent"
	"github.com/drone/drone/build/docker"
	"github.com/drone/drone/model"
	"github.com/drone/drone/yaml"

	"github.com/codegangsta/cli"
	"github.com/joho/godotenv"
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
			Name:   "secret",
			Usage:  "build secrets in KEY=VALUE format",
			EnvVar: "DRONE_SECRET",
		},
		cli.StringFlag{
			Name:   "secrets-file",
			Usage:  "build secrets file in KEY=VALUE format",
			EnvVar: "DRONE_SECRETS_FILE",
		},
		cli.StringSliceFlag{
			Name:   "matrix",
			Usage:  "build matrix in KEY=VALUE format",
			EnvVar: "DRONE_MATRIX",
		},
		cli.DurationFlag{
			Name:   "timeout",
			Usage:  "build timeout",
			Value:  time.Hour,
			EnvVar: "DRONE_TIMEOUT",
		},
		cli.DurationFlag{
			Name:   "timeout.inactivity",
			Usage:  "build timeout for inactivity",
			Value:  time.Minute * 15,
			EnvVar: "DRONE_TIMEOUT_INACTIVITY",
		},
		cli.BoolFlag{
			EnvVar: "DRONE_PLUGIN_PULL",
			Name:   "pull",
			Usage:  "always pull latest plugin images",
		},
		cli.StringSliceFlag{
			EnvVar: "DRONE_PLUGIN_PRIVILEGED",
			Name:   "privileged",
			Usage:  "plugins that require privileged mode",
			Value: &cli.StringSlice{
				"plugins/docker",
				"plugins/docker:*",
				"plugins/gcr",
				"plugins/gcr:*",
				"plugins/ecr",
				"plugins/ecr:*",
			},
		},

		// Docker daemon flags

		cli.StringFlag{
			EnvVar: "DOCKER_HOST",
			Name:   "docker-host",
			Usage:  "docker daemon address",
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
		cli.BoolTFlag{
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
		cli.BoolTFlag{
			Name:   "yaml.verified",
			Usage:  "build yaml is verified",
			EnvVar: "DRONE_YAML_VERIFIED",
		},
		cli.BoolTFlag{
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
	sigterm := make(chan os.Signal, 1)
	cancelc := make(chan bool, 1)
	signal.Notify(sigterm, os.Interrupt)
	go func() {
		<-sigterm
		cancelc <- true
	}()

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

	engine, err := docker.New(
		c.String("docker-host"),
		c.String("docker-cert-path"),
		c.Bool("docker-tls-verify"),
	)
	if err != nil {
		return err
	}

	a := agent.Agent{
		Update:   agent.NoopUpdateFunc,
		Logger:   agent.TermLoggerFunc,
		Engine:   engine,
		Timeout:  c.Duration("timeout.inactivity"),
		Platform: "linux/amd64",
		Escalate: c.StringSlice("privileged"),
		Netrc:    []string{},
		Local:    dir,
		Pull:     c.Bool("pull"),
	}

	payload := &model.Work{
		Yaml:     string(file),
		Verified: c.BoolT("yaml.verified"),
		Signed:   c.BoolT("yaml.signed"),
		Repo: &model.Repo{
			FullName:  c.String("repo.fullname"),
			Owner:     c.String("repo.owner"),
			Name:      c.String("repo.name"),
			Kind:      c.String("repo.type"),
			Link:      c.String("repo.link"),
			Branch:    c.String("repo.branch"),
			Avatar:    c.String("repo.avatar"),
			Timeout:   int64(c.Duration("timeout").Minutes()),
			IsPrivate: c.Bool("repo.private"),
			IsTrusted: c.BoolT("repo.trusted"),
			Clone:     c.String("remote.url"),
		},
		System: &model.System{
			Link: c.GlobalString("server"),
		},
		Secrets: getSecrets(c),
		Netrc: &model.Netrc{
			Login:    c.String("netrc.username"),
			Password: c.String("netrc.password"),
			Machine:  c.String("netrc.machine"),
		},
		Build: &model.Build{
			Commit:  c.String("commit.sha"),
			Branch:  c.String("commit.branch"),
			Ref:     c.String("commit.ref"),
			Link:    c.String("commit.link"),
			Message: c.String("commit.message"),
			Author:  c.String("commit.author.name"),
			Email:   c.String("commit.author.email"),
			Avatar:  c.String("commit.author.avatar"),
			Number:  c.Int("build.number"),
			Event:   c.String("build.event"),
			Deploy:  c.String("build.deploy"),
		},
		BuildLast: &model.Build{
			Number: c.Int("prev.build.number"),
			Status: c.String("prev.build.status"),
			Commit: c.String("prev.commit.sha"),
		},
	}

	if len(c.StringSlice("matrix")) > 0 {
		p := *payload
		p.Job = &model.Job{
			Environment: getMatrix(c),
		}
		return a.Run(&p, cancelc)
	}

	axes, err := yaml.ParseMatrix(file)
	if err != nil {
		return err
	}

	if len(axes) == 0 {
		axes = append(axes, yaml.Axis{})
	}

	var jobs []*model.Job
	count := 0
	for _, axis := range axes {
		jobs = append(jobs, &model.Job{
			Number:      count,
			Environment: axis,
		})
		count++
	}

	for _, job := range jobs {
		fmt.Printf("Running Matrix job #%d\n", job.Number)
		p := *payload
		p.Job = job
		if err := a.Run(&p, cancelc); err != nil {
			return err
		}
	}

	return nil
}

// helper function to retrieve matrix variables.
func getMatrix(c *cli.Context) map[string]string {
	envs := map[string]string{}
	for _, s := range c.StringSlice("matrix") {
		parts := strings.SplitN(s, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := parts[0]
		v := parts[1]
		envs[k] = v
	}
	return envs
}

// helper function to retrieve secret variables.
func getSecrets(c *cli.Context) []*model.Secret {

	var secrets []*model.Secret

	if c.String("secrets-file") != "" {
		envs, _ := godotenv.Read(c.String("secrets-file"))
		for k, v := range envs {
			secret := &model.Secret{
				Name:  k,
				Value: v,
				Events: []string{
					model.EventPull,
					model.EventPush,
					model.EventTag,
					model.EventDeploy,
				},
				Images: []string{"*"},
			}
			secrets = append(secrets, secret)
		}
	}

	for _, s := range c.StringSlice("secret") {
		parts := strings.SplitN(s, "=", 2)
		if len(parts) != 2 {
			continue
		}
		secret := &model.Secret{
			Name:  parts[0],
			Value: parts[1],
			Events: []string{
				model.EventPull,
				model.EventPush,
				model.EventTag,
				model.EventDeploy,
			},
			Images: []string{"*"},
		}
		secrets = append(secrets, secret)
	}
	return secrets
}
