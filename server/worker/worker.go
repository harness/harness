package worker

import (
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/drone/drone/plugin/notify"
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
	builds  database.BuildManager
	//config  database.ConfigManager
	pubsub *pubsub.PubSub
	server *model.Server

	request  chan *model.Request
	dispatch chan chan *model.Request
	quit     chan bool
}

func NewWorker(dispatch chan chan *model.Request, users database.UserManager, repos database.RepoManager, commits database.CommitManager, builds database.BuildManager /*config database.ConfigManager,*/, pubsub *pubsub.PubSub, server *model.Server) Worker {
	return &worker{
		users:   users,
		repos:   repos,
		commits: commits,
		builds:  builds,
		//config:   config,
		pubsub:   pubsub,
		server:   server,
		dispatch: dispatch,
		request:  make(chan *model.Request),
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
				r.Server = w.server
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
func (w *worker) Execute(r *model.Request) {
	// mark the build as Started and update the database
	r.Commit.Status = model.StatusStarted
	r.Commit.Started = time.Now().UTC().Unix()
	w.commits.Update(r.Commit)

	// parse the parameters and build script. The script has already
	// been parsed in the hook, so we can be confident it will succeed.
	// that being said, we should clean this up
	params, err := r.Repo.ParamMap()
	if err != nil {
		log.Printf("Error parsing PARAMS for %s/%s, Err: %s", r.Repo.Owner, r.Repo.Name, err.Error())
	}
	script, err := script.ParseBuild(r.Commit.Config, params)
	if err != nil {
		log.Printf("Error parsing YAML for %s/%s, Err: %s", r.Repo.Owner, r.Repo.Name, err.Error())
	}

	// append private parameters to the environment
	// variable section of the .drone.yml file, iff
	// this is not a pull request (for security purposes)
	if params != nil && (r.Repo.Private || len(r.Commit.PullRequest) == 0) {
		for k, v := range params {
			script.Env = append(script.Env, k+"="+v)
		}
	}

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

	// Instantiate a new Docker client
	var dockerClient *docker.Client
	switch {
	case len(w.server.Host) == 0:
		dockerClient = docker.New()
	default:
		dockerClient = docker.NewHost(w.server.Host)
	}

	// notify all listeners that the build is started
	commitc := w.pubsub.Register("_global")
	commitc.Publish(r)

	// send all "started" notifications
	if script.Notifications == nil {
		script.Notifications = &notify.Notification{}
	}
	script.Notifications.Send(r)

	// create an instance of the Docker builder
	var names []string
	var wg sync.WaitGroup
	wg.Add(len(r.Builds))

	for _, b := range r.Builds {
		current_index := b.Index - 1
		current_build := b

		names = append(names, b.Name)

		current_build.Status = model.StatusStarted
		current_build.Started = time.Now().UTC().Unix()
		w.builds.Update(current_build)
		go func() {
			stdoutc := w.pubsub.RegisterOpts(current_build.ID, pubsub.ConsoleOpts)
			defer stdoutc.Close()

			// create a special buffer that will also
			// write to a websocket channel
			buf := pubsub.NewBuffer(stdoutc)

			builder := build.New(dockerClient)
			builder.Index = int(current_index)
			builder.Build = script
			builder.Repo = repo
			builder.Stdout = buf
			builder.Key = []byte(r.Repo.PrivateKey)
			builder.Timeout = time.Duration(r.Repo.Timeout) * time.Second
			builder.Privileged = r.Repo.Privileged

			// run the build
			err = builder.Run()

			switch {
			case err != nil:
				current_build.Status = model.StatusError
				log.Printf("Error building %s, Commit: %s, Err: %s", current_build.Name, r.Commit.Sha, err)
				buf.WriteString(err.Error())
			case builder.BuildState == nil:
				current_build.Status = model.StatusFailure
			case builder.BuildState.ExitCode != 0:
				current_build.Status = model.StatusFailure
			default:
				current_build.Status = model.StatusSuccess
			}

			if current_build.Status != model.StatusSuccess && current_build.AllowFail {
				current_build.Status = model.StatusAllowFailure
			}

			current_build.Finished = time.Now().UTC().Unix()
			current_build.Duration = (current_build.Finished - current_build.Started)
			current_build.Output = string(buf.Bytes())

			err := w.builds.Update(current_build)
			if err != nil {
				log.Printf("Error saving result: %s, Err: %s", current_build.Name, err)
			}

			wg.Done()
		}()
	}

	// update the build status based on the results
	// from the build runner.
	wg.Wait()

	builds, err := w.builds.FindCommit(r.Commit.ID)
	if err != nil {
		r.Commit.Status = model.StatusError
		log.Printf("Error building %s, Err: %s", r.Commit.Sha, err)
	} else {
		for _, build := range builds {
			r.Commit.Status = build.Status

			if r.Commit.Status != model.StatusSuccess && build.Status != model.StatusAllowFailure {
				break
			}
		}
	}

	// calcualte the build finished and duration details and
	// update the commit
	r.Commit.Finished = time.Now().UTC().Unix()
	r.Commit.Duration = (r.Commit.Finished - r.Commit.Started)
	w.commits.Update(r.Commit)

	// notify all listeners that the build is finished
	commitc.Publish(r)

	// send all "finished" notifications
	script.Notifications.Send(r)
	log.Printf("Builds (%s) for commit (%s) finished, with %s status", strings.Join(names, ", "), r.Commit.ShaShort(), r.Commit.Status)
}
