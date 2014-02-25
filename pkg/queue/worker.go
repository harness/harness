package queue

import (
	"bytes"
	"fmt"
	"github.com/drone/drone/pkg/build"
	"github.com/drone/drone/pkg/build/git"
	r "github.com/drone/drone/pkg/build/repo"
	"github.com/drone/drone/pkg/channel"
	"github.com/drone/drone/pkg/database"
	. "github.com/drone/drone/pkg/model"
	"github.com/drone/drone/pkg/plugin/notify"
	"github.com/drone/go-github/github"
	"io"
	"log"
	"path/filepath"
	"time"
)

type worker struct{}

// work is a function that will infinitely
// run in the background waiting for tasks that
// it can pull off the queue and execute.
func (w *worker) work(queue <-chan *BuildTask) {
	var task *BuildTask
	for {
		// get work item (pointer) from the queue
		task = <-queue
		if task == nil {
			continue
		}

		// execute the task
		w.execute(task)
	}
}

// execute will execute the build task and persist
// the results to the datastore.
func (w *worker) execute(task *BuildTask) error {
	// we need to be sure that we can recover
	// from any sort panic that could occur
	// to avoid brining down the entire application
	defer func() {
		if e := recover(); e != nil {
			task.Build.Finished = time.Now().UTC()
			task.Commit.Finished = time.Now().UTC()
			task.Build.Duration = task.Build.Finished.Unix() - task.Build.Started.Unix()
			task.Commit.Duration = task.Build.Finished.Unix() - task.Build.Started.Unix()
			task.Commit.Status = "Error"
			task.Build.Status = "Error"
			database.SaveBuild(task.Build)
			database.SaveCommit(task.Commit)
		}
	}()

	// update commit and build status
	task.Commit.Status = "Started"
	task.Build.Status = "Started"
	task.Build.Started = time.Now().UTC()
	task.Commit.Started = time.Now().UTC()

	// persist the commit to the database
	if err := database.SaveCommit(task.Commit); err != nil {
		return err
	}

	// persist the build to the database
	if err := database.SaveBuild(task.Build); err != nil {
		return err
	}

	// get settings
	settings, _ := database.GetSettings()

	// notification context
	context := &notify.Context{
		Repo:   task.Repo,
		Commit: task.Commit,
		Host:   settings.URL().String(),
	}

	// send all "started" notifications
	if task.Script.Notifications != nil {
		task.Script.Notifications.Send(context)
	}

	// Send "started" notification to Github
	if err := updateGitHubStatus(task.Repo, task.Commit); err != nil {
		log.Printf("error updating github status: %s\n", err.Error())
	}

	// make sure a channel exists for the repository,
	// the commit, and the commit output (TODO)
	reposlug := fmt.Sprintf("%s/%s/%s", task.Repo.Host, task.Repo.Owner, task.Repo.Name)
	commitslug := fmt.Sprintf("%s/%s/%s/commit/%s", task.Repo.Host, task.Repo.Owner, task.Repo.Name, task.Commit.Hash)
	consoleslug := fmt.Sprintf("%s/%s/%s/commit/%s/builds/%s", task.Repo.Host, task.Repo.Owner, task.Repo.Name, task.Commit.Hash, task.Build.Slug)
	channel.Create(reposlug)
	channel.Create(commitslug)
	channel.CreateStream(consoleslug)

	// notify the channels that the commit and build started
	channel.SendJSON(reposlug, task.Commit)
	channel.SendJSON(commitslug, task.Build)

	var buf = &bufferWrapper{channel: consoleslug}

	// append private parameters to the environment
	// variable section of the .drone.yml file
	if task.Repo.Params != nil {
		for k, v := range task.Repo.Params {
			task.Script.Env = append(task.Script.Env, k+"="+v)
		}
	}

	defer func() {
		// update the status of the commit using the
		// GitHub status API.
		if err := updateGitHubStatus(task.Repo, task.Commit); err != nil {
			log.Printf("error updating github status: %s\n", err.Error())
		}
	}()

	// execute the build
	passed, buildErr := runBuild(task, buf) //w.builder.Build(script, repo, task.Repo.PrivateKey, buf)

	task.Build.Finished = time.Now().UTC()
	task.Commit.Finished = time.Now().UTC()
	task.Build.Duration = task.Build.Finished.UnixNano() - task.Build.Started.UnixNano()
	task.Commit.Duration = task.Build.Finished.UnixNano() - task.Build.Started.UnixNano()
	task.Commit.Status = "Success"
	task.Build.Status = "Success"
	task.Build.Stdout = buf.buf.String()

	// if exit code != 0 set to failure
	if passed {
		task.Commit.Status = "Failure"
		task.Build.Status = "Failure"
		if buildErr != nil && task.Build.Stdout == "" {
			// TODO: If you wanted to have very friendly error messages, you could do that here
			task.Build.Stdout = buildErr.Error() + "\n"
		}
	}

	// persist the build to the database
	if err := database.SaveBuild(task.Build); err != nil {
		return err
	}

	// persist the commit to the database
	if err := database.SaveCommit(task.Commit); err != nil {
		return err
	}

	// notify the channels that the commit and build finished
	channel.SendJSON(reposlug, task.Commit)
	channel.SendJSON(commitslug, task.Build)
	channel.Close(consoleslug)

	// send all "finished" notifications
	if task.Script.Notifications != nil {
		task.Script.Notifications.Send(context)
	}

	return nil
}

func runBuild(b *BuildTask, buf io.Writer) (bool, error) {
	builder := build.New()
	builder.Build = b.Script
	builder.Repo = &r.Repo{Path: b.Repo.URL, Branch: b.Commit.Branch, Commit: b.Commit.Hash, PR: b.Commit.PullRequest, Dir: filepath.Join("/var/cache/drone/src", b.Repo.Slug), Depth: git.GitDepth(b.Script.Git)}
	builder.Key = []byte(b.Repo.PrivateKey)
	builder.Stdout = buf
	builder.Timeout = 300 * time.Minute

	buildErr := builder.Run()

	return builder.BuildState == nil || builder.BuildState.ExitCode != 0, buildErr
}

// updateGitHubStatus is a helper function that will send
// the build status to GitHub using the Status API.
// see https://github.com/blog/1227-commit-status-api
func updateGitHubStatus(repo *Repo, commit *Commit) error {

	// convert from drone status to github status
	var message, status string
	switch commit.Status {
	case "Success":
		status = "success"
		message = "The build succeeded on drone.io"
	case "Failure":
		status = "failure"
		message = "The build failed on drone.io"
	case "Started":
		status = "pending"
		message = "The build is pending on drone.io"
	default:
		status = "error"
		message = "The build errored on drone.io"
	}

	// get the system settings
	settings, _ := database.GetSettings()

	// get the user from the database
	// since we need his / her GitHub token
	user, err := database.GetUser(repo.UserID)
	if err != nil {
		return err
	}

	client := github.New(user.GithubToken)
	client.ApiUrl = settings.GitHubApiUrl

	var url string
	url = settings.URL().String() + "/" + repo.Slug + "/commit/" + commit.Hash

	return client.Repos.CreateStatus(repo.Owner, repo.Name, status, url, message, commit.Hash)
}

type bufferWrapper struct {
	buf bytes.Buffer

	// name of the channel
	channel string
}

func (b *bufferWrapper) Write(p []byte) (n int, err error) {
	n, err = b.buf.Write(p)
	channel.SendBytes(b.channel, p)
	return
}
