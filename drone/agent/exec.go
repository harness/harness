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
	"github.com/drone/drone/engine/runner/docker"
	"github.com/drone/drone/model"
	"github.com/drone/drone/queue"
	"github.com/drone/drone/yaml/expander"

	"github.com/samalba/dockerclient"
	"golang.org/x/net/context"
)

type config struct {
	platform   string
	namespace  string
	whitelist  []string
	privileged []string
	netrc      []string
	pull       bool
}

type pipeline struct {
	drone  client.Client
	docker dockerclient.Client
	config config
}

func (r *pipeline) run() error {
	w, err := r.drone.Pull("linux", "amd64")
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
	var secrets []*model.Secret
	if w.Verified {
		secrets = append(secrets, w.Secrets...)
	}

	if w.Repo.IsPrivate {
		secrets = append(secrets, &model.Secret{
			Name:   "DRONE_NETRC_USERNAME",
			Value:  w.Netrc.Login,
			Images: []string{"*"},
			Events: []string{"*"},
		})
		secrets = append(secrets, &model.Secret{
			Name:   "DRONE_NETRC_PASSWORD",
			Value:  w.Netrc.Password,
			Images: []string{"*"},
			Events: []string{"*"},
		})
		secrets = append(secrets, &model.Secret{
			Name:   "DRONE_NETRC_MACHINE",
			Value:  w.Netrc.Machine,
			Images: []string{"*"},
			Events: []string{"*"},
		})
	}

	var lastStatus string
	if w.BuildLast != nil {
		lastStatus = w.BuildLast.Status
	}

	trans := []compiler.Transform{
		builtin.NewCloneOp("git", true),
		builtin.NewCacheOp(
			"plugins/cache:latest",
			"/var/lib/drone/cache/"+w.Repo.FullName,
			false,
		),
		builtin.NewSecretOp(w.Build.Event, secrets),
		builtin.NewNormalizeOp(r.config.namespace),
		builtin.NewWorkspaceOp("/drone", "/drone/src/github.com/"+w.Repo.FullName),
		builtin.NewValidateOp(
			w.Repo.IsTrusted,
			r.config.whitelist,
		),
		builtin.NewEnvOp(envs),
		builtin.NewShellOp(builtin.Linux_adm64),
		builtin.NewArgsOp(),
		builtin.NewEscalateOp(r.config.privileged),
		builtin.NewPodOp(prefix),
		builtin.NewAliasOp(prefix),
		builtin.NewPullOp(r.config.pull),
		builtin.NewFilterOp(
			lastStatus,
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
		w.Job.Error = err.Error()
		w.Job.ExitCode = 255
		w.Job.Finished = w.Job.Started
		w.Job.Status = model.StatusError
		pushRetry(r.drone, w)
		return nil
	}

	pushRetry(r.drone, w)

	conf := runner.Config{
		Engine: docker.New(r.docker),
	}

	c := context.TODO()
	c, timout := context.WithTimeout(c, time.Minute*time.Duration(w.Repo.Timeout))
	c, cancel := context.WithCancel(c)
	defer cancel()
	defer timout()

	run := conf.Runner(c, spec)
	run.Run()

	wait := r.drone.Wait(w.Job.ID)
	defer wait.Cancel()
	go func() {
		if _, err := wait.Done(); err == nil {
			logrus.Infof("Cancel build %s/%s#%d.%d",
				w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number)
			cancel()
		}
	}()

	rc, wc := io.Pipe()
	go func() {
		// TODO(bradrydzewski) figure out how to resume upload on failure
		err := r.drone.Stream(w.Job.ID, rc)
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

	pushRetry(r.drone, w)

	logrus.Infof("Finished build %s/%s#%d.%d",
		w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number)

	return nil
}

func pushRetry(client client.Client, w *queue.Work) {
	for {
		err := client.Push(w)
		if err == nil {
			return
		}
		logrus.Errorf("Error updating %s/%s#%d.%d. Retry in 30s. %s",
			w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number, err)
		logrus.Infof("Retry update in 30s")
		time.Sleep(time.Second * 30)
	}
}

func toEnv(w *queue.Work) map[string]string {
	envs := map[string]string{
		"CI":                         "drone",
		"DRONE":                      "true",
		"DRONE_ARCH":                 "linux_amd64",
		"DRONE_REPO":                 w.Repo.FullName,
		"DRONE_REPO_SCM":             w.Repo.Kind,
		"DRONE_REPO_OWNER":           w.Repo.Owner,
		"DRONE_REPO_NAME":            w.Repo.Name,
		"DRONE_REPO_LINK":            w.Repo.Link,
		"DRONE_REPO_AVATAR":          w.Repo.Avatar,
		"DRONE_REPO_BRANCH":          w.Repo.Branch,
		"DRONE_REPO_PRIVATE":         fmt.Sprintf("%v", w.Repo.IsPrivate),
		"DRONE_REPO_TRUSTED":         fmt.Sprintf("%v", w.Repo.IsTrusted),
		"DRONE_REMOTE_URL":           w.Repo.Clone,
		"DRONE_COMMIT_SHA":           w.Build.Commit,
		"DRONE_COMMIT_REF":           w.Build.Ref,
		"DRONE_COMMIT_BRANCH":        w.Build.Branch,
		"DRONE_COMMIT_LINK":          w.Build.Link,
		"DRONE_COMMIT_MESSAGE":       w.Build.Message,
		"DRONE_COMMIT_AUTHOR":        w.Build.Author,
		"DRONE_COMMIT_AUTHOR_EMAIL":  w.Build.Email,
		"DRONE_COMMIT_AUTHOR_AVATAR": w.Build.Avatar,
		"DRONE_BUILD_NUMBER":         fmt.Sprintf("%d", w.Build.Number),
		"DRONE_BUILD_EVENT":          w.Build.Event,
		"DRONE_BUILD_STATUS":         w.Build.Status,
		"DRONE_BUILD_LINK":           fmt.Sprintf("%s/%s/%d", w.System.Link, w.Repo.FullName, w.Build.Number),
		"DRONE_BUILD_CREATED":        fmt.Sprintf("%d", w.Build.Created),
		"DRONE_BUILD_STARTED":        fmt.Sprintf("%d", w.Build.Started),
		"DRONE_BUILD_FINISHED":       fmt.Sprintf("%d", w.Build.Finished),
		"DRONE_YAML_VERIFIED":        fmt.Sprintf("%v", w.Verified),
		"DRONE_YAML_SIGNED":          fmt.Sprintf("%v", w.Signed),
		"DRONE_BRANCH":               w.Build.Branch,
		"DRONE_COMMIT":               w.Build.Commit,
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
