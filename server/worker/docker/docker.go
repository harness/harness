package docker

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime/debug"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"code.google.com/p/go.net/context"
	"github.com/drone/drone/plugin/notify"
	"github.com/drone/drone/server/blobstore"
	"github.com/drone/drone/server/datastore"
	"github.com/drone/drone/server/pubsub"
	"github.com/drone/drone/server/worker"
	"github.com/drone/drone/shared/build"
	"github.com/drone/drone/shared/build/docker"
	"github.com/drone/drone/shared/build/git"
	"github.com/drone/drone/shared/build/repo"
	"github.com/drone/drone/shared/build/script"
	"github.com/drone/drone/shared/model"
)

const dockerKind = "docker"

type Docker struct {
	UUID    string `json:"uuid"`
	Kind    string `json:"type"`
	Created int64  `json:"created"`

	docker *docker.Client
}

func New() *Docker {
	return &Docker{
		UUID:    uuid.New(),
		Kind:    dockerKind,
		Created: time.Now().UTC().Unix(),
		docker:  docker.New(),
	}
}

func NewHost(host string) *Docker {
	return &Docker{
		UUID:    uuid.New(),
		Kind:    dockerKind,
		Created: time.Now().UTC().Unix(),
		docker:  docker.NewHost(host),
	}
}

func NewHostCertFile(host, cert, key string) *Docker {
	docker_node, err := docker.NewHostCertFile(host, cert, key)
	if err != nil {
		log.Fatalln(err)
	}

	return &Docker{
		UUID:    uuid.New(),
		Kind:    dockerKind,
		Created: time.Now().UTC().Unix(),
		docker:  docker_node,
	}
}

func (d *Docker) Do(c context.Context, r *worker.Work) {

	// ensure that we can recover from any panics to
	// avoid bringing down the entire application.
	defer func() {
		if e := recover(); e != nil {
			log.Printf("%s: %s", e, debug.Stack())
		}
	}()

	// mark the build as Started and update the database
	r.Commit.Status = model.StatusStarted
	r.Commit.Started = time.Now().UTC().Unix()

	datastore.PutCommit(c, r.Commit)

	// notify all listeners that the build is started
	commitc := pubsub.Register(c, "_global")
	commitc.Publish(r)
	stdoutc := pubsub.RegisterOpts(c, r.Commit.ID, pubsub.ConsoleOpts)
	defer pubsub.Unregister(c, r.Commit.ID)

	// create a special buffer that will also
	// write to a websocket channel
	buf := pubsub.NewBuffer(stdoutc)

	// parse the parameters and build script. The script has already
	// been parsed in the hook, so we can be confident it will succeed.
	// that being said, we should clean this up
	params, err := r.Repo.ParamMap()
	if err != nil {
		log.Printf("Error parsing PARAMS for %s/%s, Err: %s", r.Repo.Owner, r.Repo.Name, err.Error())
	}
	script, err := script.ParseBuild(script.Inject(r.Commit.Config, params))
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

	// TODO: handle error better?
	buildNumber, err := datastore.GetBuildNumber(c, r.Commit)
	if err != nil {
		log.Printf("Unable to fetch build number, Err: %s", err.Error())
	}
	script.Env = append(script.Env, fmt.Sprintf("DRONE_BUILD_NUMBER=%d", buildNumber))
	script.Env = append(script.Env, fmt.Sprintf("CI_BUILD_NUMBER=%d", buildNumber))

	path := r.Repo.Host + "/" + r.Repo.Owner + "/" + r.Repo.Name
	var branch = r.Commit.Branch
	if len(branch) == 0 {
		branch = r.Repo.DefaultBranch()
	}
	repo := &repo.Repo{
		Name:   path,
		Path:   r.Repo.CloneURL,
		Scm:    r.Repo.Scm,
		Branch: branch,
		Commit: r.Commit.Sha,
		PR:     r.Commit.PullRequest,
		Dir:    filepath.Join("/var/cache/drone/src", git.GitPath(script.Git, path)),
		Depth:  git.GitDepth(script.Git),
	}

	priorCommit, _ := datastore.GetCommitPrior(c, r.Commit)

	// send all "started" notifications
	if script.Notifications == nil {
		script.Notifications = &notify.Notification{}
	}
	script.Notifications.Send(&model.Request{
		User:   r.User,
		Repo:   r.Repo,
		Commit: r.Commit,
		Host:   r.Host,
		Prior:  priorCommit,
	})

	// create an instance of the Docker builder
	builder := build.New(d.docker)
	builder.Build = script
	builder.Repo = repo
	builder.Stdout = buf
	builder.Timeout = time.Duration(r.Repo.Timeout) * time.Second
	builder.Privileged = r.Repo.Privileged

	if r.Repo.Private || len(r.Commit.PullRequest) == 0 {
		builder.Key = []byte(r.Repo.PrivateKey)
	}

	// run the build
	err = builder.Run()

	// update the build status based on the results
	// from the build runner.
	switch {
	case err != nil:
		r.Commit.Status = model.StatusError
		log.Printf("Error building %s, Err: %s", r.Commit.Sha, err)
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
	r.Commit.Finished = time.Now().UTC().Unix()
	r.Commit.Duration = (r.Commit.Finished - r.Commit.Started)
	datastore.PutCommit(c, r.Commit)
	blobstore.Put(c, filepath.Join(r.Repo.Host, r.Repo.Owner, r.Repo.Name, r.Commit.Branch, r.Commit.Sha), buf.Bytes())

	// notify all listeners that the build is finished
	commitc.Publish(r)

	priorCommit, _ = datastore.GetCommitPrior(c, r.Commit)

	// send all "finished" notifications
	script.Notifications.Send(&model.Request{
		User:   r.User,
		Repo:   r.Repo,
		Commit: r.Commit,
		Host:   r.Host,
		Prior:  priorCommit,
	})
}
