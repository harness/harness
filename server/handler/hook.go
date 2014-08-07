package handler

import (
	"log"
	"net/http"
	"strings"

	"github.com/drone/drone/plugin/remote"
	"github.com/drone/drone/server/database"
	"github.com/drone/drone/shared/build/script"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/model"
	"github.com/gorilla/pat"
)

type HookHandler struct {
	users   database.UserManager
	repos   database.RepoManager
	commits database.CommitManager
	remotes database.RemoteManager
	queue   chan *model.Request
}

func NewHookHandler(users database.UserManager, repos database.RepoManager, commits database.CommitManager, remotes database.RemoteManager, queue chan *model.Request) *HookHandler {
	return &HookHandler{users, repos, commits, remotes, queue}
}

// PostHook receives a post-commit hook from GitHub, Bitbucket, etc
// GET /hook/:host
func (h *HookHandler) PostHook(w http.ResponseWriter, r *http.Request) error {
	host := r.FormValue(":host")
	log.Println("received post-commit hook.")

	remoteServer, err := h.remotes.FindType(host)
	if err != nil {
		return notFound{err}
	}

	remotePlugin, ok := remote.Lookup(remoteServer.Type)
	if !ok {
		return notFound{}
	}

	// get the remote system's client.
	plugin := remotePlugin(remoteServer)

	// parse the hook payload
	hook, err := plugin.GetHook(r)
	if err != nil {
		return badRequest{err}
	}

	// in some cases we have neither a hook nor error. An example
	// would be GitHub sending a ping request to the URL, in which
	// case we'll just exit quiely with an 'OK'
	if hook == nil || strings.Contains(hook.Message, "[CI SKIP]") {
		w.WriteHeader(http.StatusOK)
		return nil
	}

	// fetch the repository from the database
	repo, err := h.repos.FindName(plugin.GetHost(), hook.Owner, hook.Repo)
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
	client := plugin.GetClient(user.Access, user.Secret)
	yml, err := client.GetScript(hook)
	if err != nil {
		return badRequest{err}
	}

	// verify the commit hooks branch matches the list of approved
	// branches (unless it is a pull request). Note that we don't really
	// care if parsing the yaml fails here.
	s, _ := script.ParseBuild(yml, map[string]string{})
	if len(hook.PullRequest) == 0 && !s.MatchBranch(hook.Branch) {
		w.WriteHeader(http.StatusOK)
		return nil
	}

	c := model.Commit{
		RepoID:      repo.ID,
		Status:      model.StatusEnqueue,
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

	//fmt.Printf("%s", yml)
	owner, err := h.users.Find(repo.UserID)
	if err != nil {
		return badRequest{err}
	}

	// drop the items on the queue
	go func() {
		h.queue <- &model.Request{
			User:   owner,
			Host:   httputil.GetURL(r),
			Repo:   repo,
			Commit: &c,
		}
	}()

	w.WriteHeader(http.StatusOK)
	return nil
}

func (h *HookHandler) Register(r *pat.Router) {
	r.Post("/v1/hook/{host}", errorHandler(h.PostHook))
	r.Put("/v1/hook/{host}", errorHandler(h.PostHook))
}
