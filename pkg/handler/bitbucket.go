package handler

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/drone/drone/pkg/database"
	. "github.com/drone/drone/pkg/model"
	"github.com/drone/drone/pkg/queue"
	"github.com/drone/go-bitbucket/bitbucket"
)

type BitbucketHandler struct {
	queue *queue.Queue
}

func NewBitbucketHandler(queue *queue.Queue) *BitbucketHandler {
	return &BitbucketHandler{
		queue: queue,
	}
}

// Processes a generic POST-RECEIVE Bitbucket hook and
// attempts to trigger a build.
func (h *BitbucketHandler) Hook(w http.ResponseWriter, r *http.Request) error {
	// get the payload from the request
	payload := r.FormValue("payload")

	// parse the post-commit hook
	hook, err := bitbucket.ParseHook([]byte(payload))
	if err != nil {
		return err
	}

	// get the repo from the URL
	repoId := r.FormValue("id")

	// get the repo from the database, return error if not found
	repo, err := database.GetRepoSlug(repoId)
	if err != nil {
		return RenderText(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}

	// Get the user that owns the repository
	user, err := database.GetUser(repo.UserID)
	if err != nil {
		return RenderText(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	}

	// Verify that the commit doesn't already exist.
	// We should never build the same commit twice.
	_, err = database.GetCommitHash(hook.Commits[len(hook.Commits)-1].Hash, repo.ID)
	if err != nil && err != sql.ErrNoRows {
		return RenderText(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
	}

	commit := &Commit{}
	commit.RepoID = repo.ID
	commit.Branch = hook.Commits[len(hook.Commits)-1].Branch
	commit.Hash = hook.Commits[len(hook.Commits)-1].Hash
	commit.Status = "Pending"
	commit.Created = time.Now().UTC()
	commit.Message = hook.Commits[len(hook.Commits)-1].Message
	commit.Timestamp = time.Now().UTC().String()
	commit.SetAuthor(hook.Commits[len(hook.Commits)-1].Author)

	// get the github settings from the database
	settings := database.SettingsMust()

	// create the Bitbucket client
	client := bitbucket.New(
		settings.BitbucketKey,
		settings.BitbucketSecret,
		user.BitbucketToken,
		user.BitbucketSecret,
	)

	// get the yaml from the database
	raw, err := client.Sources.Find(repo.Owner, repo.Name, commit.Hash, ".drone.yml")
	if err != nil {
		return RenderText(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}

	// save the commit to the database
	if err := database.SaveCommit(commit); err != nil {
		return RenderText(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	// save the build to the database
	build := &Build{}
	build.Slug = "1" // TODO
	build.CommitID = commit.ID
	build.Created = time.Now().UTC()
	build.Status = "Pending"
	build.BuildScript = raw.Data
	if err := database.SaveBuild(build); err != nil {
		return RenderText(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	// send the build to the queue
	h.queue.Add(&queue.BuildTask{Repo: repo, Commit: commit, Build: build})

	// OK!
	return RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
}
