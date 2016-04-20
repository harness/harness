package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/dchest/uniuri"
	"github.com/drone/drone/client"
	"github.com/drone/drone/engine/compiler"
	"github.com/drone/drone/engine/compiler/builtin"
	"github.com/drone/drone/engine/runner"
	engine "github.com/drone/drone/engine/runner/docker"
	"github.com/drone/drone/model"

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

	trans := []compiler.Transform{
		builtin.NewCloneOp("plugins/git:latest", true),
		builtin.NewCacheOp(
			"plugins/cache:latest",
			"/var/lib/drone/cache/"+w.Repo.FullName,
			false,
		),
		builtin.NewNormalizeOp("plugins"),
		builtin.NewWorkspaceOp("/drone", "drone/src/github.com/"+w.Repo.FullName),
		builtin.NewEnvOp(map[string]string{
			"CI":                "drone",
			"CI_REPO":           w.Repo.FullName,
			"CI_REPO_OWNER":     w.Repo.Owner,
			"CI_REPO_NAME":      w.Repo.Name,
			"CI_REPO_LINK":      w.Repo.Link,
			"CI_REPO_AVATAR":    w.Repo.Avatar,
			"CI_REPO_BRANCH":    w.Repo.Branch,
			"CI_REPO_PRIVATE":   fmt.Sprintf("%v", w.Repo.IsPrivate),
			"CI_REMOTE_URL":     w.Repo.Clone,
			"CI_COMMIT_SHA":     w.Build.Commit,
			"CI_COMMIT_REF":     w.Build.Ref,
			"CI_COMMIT_BRANCH":  w.Build.Branch,
			"CI_COMMIT_LINK":    w.Build.Link,
			"CI_COMMIT_MESSAGE": w.Build.Message,
			"CI_AUTHOR":         w.Build.Author,
			"CI_AUTHOR_EMAIL":   w.Build.Email,
			"CI_AUTHOR_AVATAR":  w.Build.Avatar,
			"CI_BUILD_NUMBER":   fmt.Sprintf("%v", w.Build.Number),
			"CI_BUILD_EVENT":    w.Build.Event,
			// "CI_NETRC_USERNAME":    w.Netrc.Login,
			// "CI_NETRC_PASSWORD":    w.Netrc.Password,
			// "CI_NETRC_MACHINE":     w.Netrc.Machine,
			// "CI_PREV_BUILD_STATUS": w.BuildLast.Status,
			// "CI_PREV_BUILD_NUMBER": fmt.Sprintf("%v", w.BuildLast.Number),
			// "CI_PREV_COMMIT_SHA":   w.BuildLast.Commit,
		}),
		builtin.NewValidateOp(
			w.Repo.IsTrusted,
			[]string{"plugins/*"},
		),
		builtin.NewShellOp(builtin.Linux_adm64),
		builtin.NewArgsOp(),
		builtin.NewPodOp(prefix),
		builtin.NewAliasOp(prefix),
		builtin.NewPullOp(false),
		builtin.NewFilterOp(
			model.StatusSuccess, // w.BuildLast.Status,
			w.Build.Branch,
			w.Build.Event,
			w.Build.Deploy,
			map[string]string{},
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
	case 128, 130:
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
