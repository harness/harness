package handler

import (
	"fmt"
	"net/http"

	"github.com/drone/drone/server/queue"
	"github.com/drone/drone/server/resource/commit"
	"github.com/drone/drone/server/resource/config"
	"github.com/drone/drone/server/resource/repo"
	"github.com/drone/drone/server/resource/user"
	"github.com/gorilla/pat"
)

type HookHandler struct {
	users   user.UserManager
	repos   repo.RepoManager
	commits commit.CommitManager
	queue   *queue.Queue
	conf    *config.Config
}

func NewHookHandler(users user.UserManager, repos repo.RepoManager, commits commit.CommitManager, conf *config.Config, queue *queue.Queue) *HookHandler {
	return &HookHandler{users, repos, commits, queue, conf}
}

// PostHook receives a post-commit hook from GitHub, Bitbucket, etc
// GET /hook/:host
func (h *HookHandler) PostHook(w http.ResponseWriter, r *http.Request) error {
	host := r.FormValue(":host")

	// get the remote system's client.
	remote := h.conf.GetRemote(host)
	if remote == nil {
		return notFound{}
	}

	// parse the hook payload
	hook, err := remote.GetHook(r)
	if err != nil {
		return badRequest{err}
	}

	// in some cases we have neither a hook nor error. An example
	// would be GitHub sending a ping request to the URL, in which
	// case we'll just exit quiely with an 'OK'
	if hook == nil {
		w.WriteHeader(http.StatusOK)
		return nil
	}

	// fetch the repository from the database
	repo, err := h.repos.FindName(remote.GetHost(), hook.Owner, hook.Repo)
	if err != nil {
		return notFound{}
	}

	if repo.Active == false ||
		(repo.PostCommit == false && len(hook.PullRequest) == 0) ||
		(repo.PullRequest == false && len(hook.PullRequest) != 0) {
		w.WriteHeader(http.StatusOK)
		return nil
	}

	// fetch the user from the database that owns this repo
	user, err := h.users.Find(repo.UserID)
	if err != nil {
		return notFound{}
	}

	// featch the .drone.yml file from the database
	client := remote.GetClient(user.Access, user.Secret)
	yml, err := client.GetScript(hook)
	if err != nil {
		return badRequest{err}
	}

	c := commit.Commit{
		RepoID:      repo.ID,
		Status:      commit.StatusEnqueue,
		Sha:         hook.Sha,
		Branch:      hook.Branch,
		PullRequest: hook.PullRequest,
		Timestamp:   hook.Timestamp,
		Message:     hook.Message,
		Config:      yml}
	c.SetAuthor(hook.Author)
	// inser the commit into the database
	if err := h.commits.Insert(&c); err != nil {
		return badRequest{err}
	}

	fmt.Printf("%#v", hook)
	fmt.Printf("%s", yml)

	// drop the items on the queue
	h.queue.Add(&queue.BuildTask{Repo: repo, Commit: &c})
	return nil
}

func (h *HookHandler) Register(r *pat.Router) {
	r.Post("/hook/{host}", errorHandler(h.PostHook))
	r.Put("/hook/{host}", errorHandler(h.PostHook))
}
