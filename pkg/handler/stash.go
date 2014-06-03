package handler

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/drone/drone/pkg/database"
	. "github.com/drone/drone/pkg/model"
	"github.com/drone/drone/pkg/queue"
	"github.com/reinbach/go-stash/stash"
)

type StashHandler struct {
	queue *queue.Queue
}

func NewStashHandler(queue *queue.Queue) *StashHandler {
	return &StashHandler{
		queue: queue,
	}
}

// Processes a generic POST-RECEIVE Stash hook and
// attempts to trigger a build.
func (h *StashHandler) Hook(w http.ResponseWriter, r *http.Request) error {
	// get the project and repo from the request
	repoId := r.FormValue("id")
	branch := r.FormValue("branch")
	hash := r.FormValue("hash")
	message := r.FormValue("type")
	author := r.FormValue("displayName")

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
	_, err = database.GetCommitHash(hash, repo.ID)
	if err != nil && err != sql.ErrNoRows {
		return RenderText(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
	}

	commit := &Commit{}
	commit.RepoID = repo.ID
	commit.Branch = branch
	commit.Hash = hash
	commit.Status = "Pending"
	commit.Created = time.Now().UTC()
	commit.Message = message
	commit.Timestamp = time.Now().UTC().String()
	commit.SetAuthor(author)

	// get the slash settings from the database
	settings := database.SettingsMust()

	// create the Stash client
	client := stash.New(
		settings.StashDomain,
		settings.StashKey,
		user.StashToken,
		user.StashSecret,
		settings.StashPrivateKey,
	)

	// get the yaml from the database
	raw, err := client.Contents.Find(repo.Owner, repo.Name, ".drone.yml")
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
	build.BuildScript = raw
	if err := database.SaveBuild(build); err != nil {
		return RenderText(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	// send the build to the queue
	h.queue.Add(&queue.BuildTask{Repo: repo, Commit: commit, Build: build})

	// OK!
	return RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
}
