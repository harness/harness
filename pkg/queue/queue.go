package queue

import (
	"bytes"
	"fmt"
	bldr "github.com/drone/drone/pkg/build"
	r "github.com/drone/drone/pkg/build/repo"
	"github.com/drone/drone/pkg/build/script"
	"github.com/drone/drone/pkg/build/script/notification"
	"github.com/drone/drone/pkg/channel"
	"github.com/drone/drone/pkg/database"
	"github.com/drone/drone/pkg/mail"
	. "github.com/drone/drone/pkg/model"
	"github.com/drone/go-github/github"
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
	context := &notification.Context{
		Repo:   b.Repo,
		Commit: b.Commit,
		Host:   settings.URL().String(),
	}

	// send all "started" notifications
	if b.Script.Notifications != nil {
		b.Script.Notifications.Send(context)
	}

	// make sure a channel exists for the repository,
	// the commit, and the commit output (TODO)
	reposlug := fmt.Sprintf("%s/%s/%s", b.Repo.Host, b.Repo.Owner, b.Repo.Name)
	commitslug := fmt.Sprintf("%s/%s/%s/commit/%s", b.Repo.Host, b.Repo.Owner, b.Repo.Name, b.Commit.Hash)
	consoleslug := fmt.Sprintf("%s/%s/%s/commit/%s/builds/%s", b.Repo.Host, b.Repo.Owner, b.Repo.Name, b.Commit.Hash, b.Build.Slug)
	channel.Create(reposlug)
	channel.Create(commitslug)
	channel.CreateStream(consoleslug)

	// notify the channels that the commit and build started
	channel.SendJSON(reposlug, b.Commit)
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
	builder.Repo = &r.Repo{Path: b.Repo.URL, Branch: b.Commit.Branch, Commit: b.Commit.Hash, PR: b.Commit.PullRequest, Dir: filepath.Join("/var/cache/drone/src", b.Repo.Slug)}
	builder.Key = []byte(b.Repo.PrivateKey)
	builder.Stdout = buf
	builder.Timeout = 300 * time.Minute
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
	channel.SendJSON(commitslug, b.Build)
	channel.Close(consoleslug)

	// add the smtp address to the notificaitons
	//if b.Script.Notifications != nil && b.Script.Notifications.Email != nil {
	//	b.Script.Notifications.Email.SetServer(settings.SmtpServer, settings.SmtpPort,
	//		settings.SmtpUsername, settings.SmtpPassword, settings.SmtpAddress)
	//}

	// send all "finished" notifications
	if b.Script.Notifications != nil {
		b.sendEmail(context) // send email from queue, not from inside /build/script package
		b.Script.Notifications.Send(context)
	}

	// update the status of the commit using the
	// GitHub status API.
	if err := updateGitHubStatus(b.Repo, b.Commit); err != nil {
		return err
	}

	return nil
}

// updateGitHubStatus is a helper function that will send
// the build status to GitHub using the Status API.
// see https://github.com/blog/1227-commit-status-api
func updateGitHubStatus(repo *Repo, commit *Commit) error {

	// convert from drone status to github status
	var message, status string
	switch status {
	case "Success":
		status = "success"
		message = "The build succeeded on drone.io"
	case "Failure":
		status = "failure"
		message = "The build failed on drone.io"
	case "Pending":
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
	if err == nil {
		return err
	}

	client := github.New(user.GithubToken)
	return client.Repos.CreateStatus(repo.Owner, repo.Name, status, settings.URL().String(), message, commit.Hash)
}

func (t *BuildTask) sendEmail(c *notification.Context) error {
	// make sure a notifications object exists
	if t.Script.Notifications == nil && t.Script.Notifications.Email != nil {
		return nil
	}

	switch {
	case t.Commit.Status == "Success" && t.Script.Notifications.Email.Success != "never":
		return t.sendSuccessEmail(c)
	case t.Commit.Status == "Failure" && t.Script.Notifications.Email.Failure != "never":
		return t.sendFailureEmail(c)
	default:
		println("sending nothing")
	}

	return nil
}

// sendFailure sends email notifications to the list of
// recipients indicating the build failed.
func (t *BuildTask) sendFailureEmail(c *notification.Context) error {

	// loop through and email recipients
	for _, email := range t.Script.Notifications.Email.Recipients {
		if err := mail.SendFailure(t.Repo.Name, email, c); err != nil {
			return err
		}
	}
	return nil
}

// sendSuccess sends email notifications to the list of
// recipients indicating the build was a success.
func (t *BuildTask) sendSuccessEmail(c *notification.Context) error {

	// loop through and email recipients
	for _, email := range t.Script.Notifications.Email.Recipients {
		if err := mail.SendSuccess(t.Repo.Name, email, c); err != nil {
			return err
		}
	}
	return nil
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
