package queue

import (
	"bytes"
	"fmt"
	bldr "github.com/drone/drone/pkg/build"
	"github.com/drone/drone/pkg/build/git"
	r "github.com/drone/drone/pkg/build/repo"
	"github.com/drone/drone/pkg/build/script"
	"github.com/drone/drone/pkg/channel"
	"github.com/drone/drone/pkg/database"
	. "github.com/drone/drone/pkg/model"
	"github.com/drone/drone/pkg/plugin/notify"
	"github.com/drone/go-github/github"
	"log"
	"path/filepath"
	"time"
)

// queue that will store all build tasks until
// they are processed by a worker.
var queue = make(chan *BuildTask)

// work is a function that will infinitely
// run in the background waiting for tasks that
// it can pull off the queue and execute.
func work() {
	var task *BuildTask
	for {
		// get work item (pointer) from the queue
		task = <-queue
		if task == nil {
			continue
		}

		// execute the task
		task.execute()
	}
}

// Add adds the task to the build queue.
func Add(task *BuildTask) {
	queue <- task
}

// BuildTasks represents a build that is pending
// execution.
type BuildTask struct {
	Repo   *Repo
	Commit *Commit
	Build  *Build

	// Build instructions from the .drone.yml
	// file, unmarshalled.
	Script *script.Build
}

// execute will execute the build task and persist
// the results to the datastore.
func (b *BuildTask) execute() error {
	// we need to be sure that we can recover
	// from any sort panic that could occur
	// to avoid brining down the entire application
	defer func() {
		if e := recover(); e != nil {
			b.Build.Finished = time.Now().UTC()
			b.Commit.Finished = time.Now().UTC()
			b.Build.Duration = b.Build.Finished.Unix() - b.Build.Started.Unix()
			b.Commit.Duration = b.Build.Finished.Unix() - b.Build.Started.Unix()
			b.Commit.Status = "Error"
			b.Build.Status = "Error"
			database.SaveBuild(b.Build)
			database.SaveCommit(b.Commit)
		}
	}()

	// update commit and build status
	b.Commit.Status = "Started"
	b.Build.Status = "Started"
	b.Build.Started = time.Now().UTC()
	b.Commit.Started = time.Now().UTC()

	// persist the commit to the database
	if err := database.SaveCommit(b.Commit); err != nil {
		return err
	}

	// persist the build to the database
	if err := database.SaveBuild(b.Build); err != nil {
		return err
	}

	// get settings
	settings, _ := database.GetSettings()

	// notification context
	context := &notify.Context{
		Repo:   b.Repo,
		Commit: b.Commit,
		Host:   settings.URL().String(),
	}

	// send all "started" notifications
	if b.Script.Notifications != nil {
		b.Script.Notifications.Send(context)
	}

	// Send "started" notification to Github
	if err := updateGitHubStatus(b.Repo, b.Commit); err != nil {
		log.Printf("error updating github status: %s\n", err.Error())
	}

	// make sure a channel exists for the repository,
	// the commit, and the commit output (TODO)
	var wallslug string
	if b.Repo.TeamID != 0 {
		wallslug = fmt.Sprintf("wall/team/%d", b.Repo.TeamID)
	} else {
		wallslug = fmt.Sprintf("wall/user/%d", b.Repo.UserID)
	}

	reposlug := fmt.Sprintf("%s/%s/%s", b.Repo.Host, b.Repo.Owner, b.Repo.Name)
	commitslug := fmt.Sprintf("%s/%s/%s/commit/%s", b.Repo.Host, b.Repo.Owner, b.Repo.Name, b.Commit.Hash)
	consoleslug := fmt.Sprintf("%s/%s/%s/commit/%s/builds/%s", b.Repo.Host, b.Repo.Owner, b.Repo.Name, b.Commit.Hash, b.Build.Slug)
	channel.Create(reposlug)
	channel.Create(commitslug)
	channel.Create(wallslug)
	channel.CreateStream(consoleslug)

	// notify the channels that the commit and build started
	channel.SendJSON(reposlug, b.Commit)
	channel.SendJSON(wallslug, b)
	channel.SendJSON(commitslug, b.Build)

	var buf = &bufferWrapper{channel: consoleslug}

	// append private parameters to the environment
	// variable section of the .drone.yml file
	if b.Repo.Params != nil {
		for k, v := range b.Repo.Params {
			b.Script.Env = append(b.Script.Env, k+"="+v)
		}
	}

	// execute the build
	builder := bldr.Builder{}
	builder.Build = b.Script
	builder.Repo = &r.Repo{Path: b.Repo.URL, Branch: b.Commit.Branch, Commit: b.Commit.Hash, PR: b.Commit.PullRequest, Dir: filepath.Join("/var/cache/drone/src", b.Repo.Slug), Depth: git.GitDepth(b.Script.Git)}
	builder.Key = []byte(b.Repo.PrivateKey)
	builder.Stdout = buf
	builder.Timeout = 300 * time.Minute

	defer func() {
		// update the status of the commit using the
		// GitHub status API.
		if err := updateGitHubStatus(b.Repo, b.Commit); err != nil {
			log.Printf("error updating github status: %s\n", err.Error())
		}
	}()

	buildErr := builder.Run()

	b.Build.Finished = time.Now().UTC()
	b.Commit.Finished = time.Now().UTC()
	b.Build.Duration = b.Build.Finished.UnixNano() - b.Build.Started.UnixNano()
	b.Commit.Duration = b.Build.Finished.UnixNano() - b.Build.Started.UnixNano()
	b.Commit.Status = "Success"
	b.Build.Status = "Success"
	b.Build.Stdout = buf.buf.String()

	// if exit code != 0 set to failure
	if builder.BuildState == nil || builder.BuildState.ExitCode != 0 {
		b.Commit.Status = "Failure"
		b.Build.Status = "Failure"
		if buildErr != nil && b.Build.Stdout == "" {
			// TODO: If you wanted to have very friendly error messages, you could do that here
			b.Build.Stdout = buildErr.Error() + "\n"
		}
	}

	// persist the build to the database
	if err := database.SaveBuild(b.Build); err != nil {
		return err
	}

	// persist the commit to the database
	if err := database.SaveCommit(b.Commit); err != nil {
		return err
	}

	// notify the channels that the commit and build finished
	channel.SendJSON(reposlug, b.Commit)
	channel.SendJSON(wallslug, b)
	channel.SendJSON(commitslug, b.Build)
	channel.Close(consoleslug)

	// send all "finished" notifications
	if b.Script.Notifications != nil {
		b.Script.Notifications.Send(context)
	}

	return nil
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
