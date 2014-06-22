package worker

import (
	"path/filepath"
	"time"

	"github.com/drone/drone/server/database"
	"github.com/drone/drone/server/pubsub"
	"github.com/drone/drone/shared/build"
	"github.com/drone/drone/shared/build/docker"
	"github.com/drone/drone/shared/build/git"
	"github.com/drone/drone/shared/build/repo"
	"github.com/drone/drone/shared/build/script"
	"github.com/drone/drone/shared/model"
)

type Worker interface {
	Start() // Start instructs the worker to start processing requests
	Stop()  // Stop instructions the worker to stop processing requests
}

type worker struct {
	users   database.UserManager
	repos   database.RepoManager
	commits database.CommitManager
	config  database.ConfigManager
	pubsub  *pubsub.PubSub
	server  *model.Server

	request  chan *Request
	dispatch chan chan *Request
	quit     chan bool
}

func NewWorker(dispatch chan chan *Request, users database.UserManager, repos database.RepoManager, commits database.CommitManager, config database.ConfigManager, pubsub *pubsub.PubSub, server *model.Server) Worker {
	return &worker{
		users:    users,
		repos:    repos,
		commits:  commits,
		config:   config,
		pubsub:   pubsub,
		server:   server,
		dispatch: dispatch,
		request:  make(chan *Request),
		quit:     make(chan bool),
	}
}

// Start tells the worker to start listening and
// accepting new work requests.
func (w *worker) Start() {
	go func() {
		for {
			// register our queue with the dispatch
			// queue to start accepting work.
			go func() { w.dispatch <- w.request }()

			select {
			case r := <-w.request:
				// handle the request
				r.server = w.server
				w.Execute(r)

			case <-w.quit:
				return
			}
		}
	}()
}

// Stop tells the worker to stop listening for new
// work requests.
func (w *worker) Stop() {
	go func() { w.quit <- true }()
}

// Execute executes the work Request, persists the
// results to the database, and sends event messages
// to the pubsub (for websocket updates on the website).
func (w *worker) Execute(r *Request) {
	// mark the build as Started and update the database
	r.Commit.Status = model.StatusStarted
	r.Commit.Started = time.Now().Unix()
	w.commits.Update(r.Commit)

	// notify all listeners that the build is started
	commitc := w.pubsub.Register("_global")
	commitc.Publish(r)
	stdoutc := w.pubsub.RegisterOpts(r.Commit.ID, pubsub.ConsoleOpts)
	defer stdoutc.Close()

	// create a special buffer that will also
	// write to a websocket channel
	buf := pubsub.NewBuffer(stdoutc)

	// parse the parameters and build script. The script has already
	// been parsed in the hook, so we can be confident it will succeed.
	// that being said, we should clean this up
	params, _ := r.Repo.ParamMap()
	script, _ := script.ParseBuild(r.Commit.Config, params)

	path := r.Repo.Host + "/" + r.Repo.Owner + "/" + r.Repo.Name
	repo := &repo.Repo{
		Name:   path,
		Path:   r.Repo.CloneURL,
		Branch: r.Commit.Branch,
		Commit: r.Commit.Sha,
		PR:     r.Commit.PullRequest,
		Dir:    filepath.Join("/var/cache/drone/src", git.GitPath(script.Git, path)),
		Depth:  git.GitDepth(script.Git),
	}

	// create an instance of the Docker builder
	builder := build.New(docker.NewHost(w.server.Host))
	builder.Build = script
	builder.Repo = repo
	builder.Stdout = buf
	builder.Key = []byte(r.Repo.PrivateKey)
	builder.Timeout = time.Duration(r.Repo.Timeout) * time.Minute
	builder.Privileged = r.Repo.Privileged

	//err := builder.Run()
	var err error
	buf.WriteString("THIS\n")
	time.Sleep(3 * time.Second)
	buf.WriteString("IS\n")
	time.Sleep(3 * time.Second)
	buf.WriteString("A\n")
	time.Sleep(3 * time.Second)
	buf.WriteString("TEST\n")
	time.Sleep(3 * time.Second)

	// update the build status based on the results
	// from the build runner.
	switch {
	case err != nil:
		r.Commit.Status = model.StatusError
		buf.WriteString(err.Error())
	case builder.BuildState == nil:
		r.Commit.Status = model.StatusFailure
	case builder.BuildState.ExitCode != 0:
		r.Commit.Status = model.StatusFailure
	default:
		r.Commit.Status = model.StatusSuccess
	}

	// calcualte the build finished and duration details and
	// update the commit
	r.Commit.Finished = time.Now().Unix()
	r.Commit.Duration = (r.Commit.Finished - r.Commit.Started)
	w.commits.Update(r.Commit)
	w.commits.UpdateOutput(r.Commit, buf.Bytes())

	// notify all listeners that the build is finished
	commitc.Publish(r)

	// todo(bradrydzewski) update github status API
	// todo(bradrydzewski) send email notifications
	// todo(bradrydzewski) send other notifications
}
