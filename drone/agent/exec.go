package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/dchest/uniuri"
	"github.com/drone/drone/client"
	"github.com/drone/drone/engine/compiler"
	"github.com/drone/drone/engine/compiler/builtin"
	"github.com/drone/drone/engine/runner"
	engine "github.com/drone/drone/engine/runner/docker"
	"github.com/drone/drone/model"
	"github.com/drone/drone/queue"
	"github.com/drone/drone/yaml/expander"

	"github.com/samalba/dockerclient"
	"golang.org/x/net/context"
)

func recoverExec(client client.Client, docker dockerclient.Client) error {
	defer func() {
		recover()
	}()
	return exec(client, docker)
}

func exec(client client.Client, docker dockerclient.Client) error {
	w, err := client.Pull()
	if err != nil {
		return err
	}

	logrus.Infof("Starting build %s/%s#%d.%d",
		w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number)

	w.Job.Status = model.StatusRunning
	w.Job.Started = time.Now().Unix()

	prefix := fmt.Sprintf("drone_%s", uniuri.New())

	envs := toEnv(w)
	w.Yaml = expander.ExpandString(w.Yaml, envs)

	// inject the netrc file into the clone plugin if the repositroy is
	// private and requires authentication.
	if w.Repo.IsPrivate {
		w.Secrets = append(w.Secrets, &model.Secret{
			Name:   "DRONE_NETRC_USERNAME",
			Value:  w.Netrc.Login,
			Images: []string{"git", "hg"}, // TODO(bradrydzewski) use the command line parameters here
			Events: []string{model.EventDeploy, model.EventPull, model.EventPush, model.EventTag},
		})
		w.Secrets = append(w.Secrets, &model.Secret{
			Name:   "DRONE_NETRC_PASSWORD",
			Value:  w.Netrc.Password,
			Images: []string{w.Repo.Kind},
			Events: []string{model.EventDeploy, model.EventPull, model.EventPush, model.EventTag},
		})
		w.Secrets = append(w.Secrets, &model.Secret{
			Name:   "DRONE_NETRC_MACHINE",
			Value:  w.Netrc.Machine,
			Images: []string{"git", "hg"},
			Events: []string{model.EventDeploy, model.EventPull, model.EventPush, model.EventTag},
		})
	}

	trans := []compiler.Transform{
		builtin.NewCloneOp("plugins/"+w.Repo.Kind+":latest", true),
		builtin.NewCacheOp(
			"plugins/cache:latest",
			"/var/lib/drone/cache/"+w.Repo.FullName,
			false,
		),
		builtin.NewSecretOp(w.Build.Event, w.Secrets),
		builtin.NewNormalizeOp("plugins"),
		builtin.NewWorkspaceOp("/drone", "drone/src/github.com/"+w.Repo.FullName),
		builtin.NewValidateOp(
			w.Repo.IsTrusted,
			[]string{"plugins/*"},
		),
		builtin.NewEnvOp(envs),
		builtin.NewShellOp(builtin.Linux_adm64),
		builtin.NewArgsOp(),
		builtin.NewPodOp(prefix),
		builtin.NewAliasOp(prefix),
		builtin.NewPullOp(false),
		builtin.NewFilterOp(
			model.StatusSuccess, // TODO(bradrydzewski) please add the last build status here
			w.Build.Branch,
			w.Build.Event,
			w.Build.Deploy,
			w.Job.Environment,
		),
	}

	compile := compiler.New()
	compile.Transforms(trans)
	spec, err := compile.CompileString(w.Yaml)
	if err != nil {
		// TODO handle error
		logrus.Infof("Error compiling Yaml %s/%s#%d %s",
			w.Repo.Owner, w.Repo.Name, w.Build.Number, err.Error())
		return err
	}

	if err := client.Push(w); err != nil {
		logrus.Errorf("Error persisting update %s/%s#%d.%d. %s",
			w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number, err)
		return err
	}

	conf := runner.Config{
		Engine: engine.New(docker),
	}

	ctx := context.TODO()
	ctx, cancel := context.WithCancel(ctx)

	run := conf.Runner(ctx, spec)
	run.Run()
	defer cancel()

	wait := client.Wait(w.Job.ID)
	if err != nil {
		return err
	}
	go func() {
		_, werr := wait.Done()
		if werr == nil {
			logrus.Infof("Cancel build %s/%s#%d.%d",
				w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number)
			cancel()
		}
	}()
	defer wait.Cancel()

	rc, wc := io.Pipe()
	go func() {
		err := client.Stream(w.Job.ID, rc)
		if err != nil && err != io.ErrClosedPipe {
			logrus.Errorf("Error streaming build logs. %s", err)
		}
	}()

	pipe := run.Pipe()
	for {
		line := pipe.Next()
		if line == nil {
			break
		}
		linejson, _ := json.Marshal(line)
		wc.Write(linejson)
		wc.Write([]byte{'\n'})
	}

	err = run.Wait()

	pipe.Close()
	wc.Close()
	rc.Close()

	// catch the build result
	if err != nil {
		w.Job.ExitCode = 255
	}
	if exitErr, ok := err.(*runner.ExitError); ok {
		w.Job.ExitCode = exitErr.Code
	}

	w.Job.Finished = time.Now().Unix()

	switch w.Job.ExitCode {
	case 128, 130, 137:
		w.Job.Status = model.StatusKilled
	case 0:
		w.Job.Status = model.StatusSuccess
	default:
		w.Job.Status = model.StatusFailure
	}

	logrus.Infof("Finished build %s/%s#%d.%d",
		w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number)

	return client.Push(w)
}

func toEnv(w *queue.Work) map[string]string {
	envs := map[string]string{
		"CI":                   "drone",
		"DRONE":                "true",
		"DRONE_ARCH":           "linux_amd64",
		"DRONE_REPO":           w.Repo.FullName,
		"DRONE_REPO_SCM":       w.Repo.Kind,
		"DRONE_REPO_OWNER":     w.Repo.Owner,
		"DRONE_REPO_NAME":      w.Repo.Name,
		"DRONE_REPO_LINK":      w.Repo.Link,
		"DRONE_REPO_AVATAR":    w.Repo.Avatar,
		"DRONE_REPO_BRANCH":    w.Repo.Branch,
		"DRONE_REPO_PRIVATE":   fmt.Sprintf("%v", w.Repo.IsPrivate),
		"DRONE_REPO_TRUSTED":   fmt.Sprintf("%v", w.Repo.IsTrusted),
		"DRONE_REMOTE_URL":     w.Repo.Clone,
		"DRONE_COMMIT_SHA":     w.Build.Commit,
		"DRONE_COMMIT_REF":     w.Build.Ref,
		"DRONE_COMMIT_BRANCH":  w.Build.Branch,
		"DRONE_COMMIT_LINK":    w.Build.Link,
		"DRONE_COMMIT_MESSAGE": w.Build.Message,
		"DRONE_AUTHOR":         w.Build.Author,
		"DRONE_AUTHOR_EMAIL":   w.Build.Email,
		"DRONE_AUTHOR_AVATAR":  w.Build.Avatar,
		"DRONE_BUILD_NUMBER":   fmt.Sprintf("%d", w.Build.Number),
		"DRONE_BUILD_EVENT":    w.Build.Event,
		"DRONE_BUILD_CREATED":  fmt.Sprintf("%d", w.Build.Created),
		"DRONE_BUILD_STARTED":  fmt.Sprintf("%d", w.Build.Started),
		"DRONE_BUILD_FINISHED": fmt.Sprintf("%d", w.Build.Finished),
		"DRONE_BUILD_VERIFIED": fmt.Sprintf("%v", false),

		// SHORTER ALIASES
		"DRONE_BRANCH": w.Build.Branch,
		"DRONE_COMMIT": w.Build.Commit,

		// TODO(bradrydzewski) netrc should only be injected via secrets
		// "DRONE_NETRC_USERNAME":    w.Netrc.Login,
		// "DRONE_NETRC_PASSWORD":    w.Netrc.Password,
		// "DRONE_NETRC_MACHINE":     w.Netrc.Machine,
	}

	if w.Build.Event == model.EventTag {
		envs["DRONE_TAG"] = strings.TrimPrefix(w.Build.Ref, "refs/tags/")
	}
	if w.Build.Event == model.EventPull {
		envs["DRONE_PULL_REQUEST"] = pullRegexp.FindString(w.Build.Ref)
	}
	if w.Build.Event == model.EventDeploy {
		envs["DRONE_DEPLOY_TO"] = w.Build.Deploy
	}

	if w.BuildLast != nil {
		envs["DRONE_PREV_BUILD_STATUS"] = w.BuildLast.Status
		envs["DRONE_PREV_BUILD_NUMBER"] = fmt.Sprintf("%v", w.BuildLast.Number)
		envs["DRONE_PREV_COMMIT_SHA"] = w.BuildLast.Commit
	}

	// inject matrix values as environment variables
	for key, val := range w.Job.Environment {
		envs[key] = val
	}
	return envs
}

var pullRegexp = regexp.MustCompile("\\d+")
