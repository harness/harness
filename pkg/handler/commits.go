package handler

import (
	"fmt"
	"net/http"

	"github.com/drone/drone/pkg/channel"
	"github.com/drone/drone/pkg/database"
	. "github.com/drone/drone/pkg/model"
	"github.com/drone/drone/pkg/queue"
)

// Display a specific Commit.
func CommitShow(w http.ResponseWriter, r *http.Request, u *User, repo *Repo) error {
	hash := r.FormValue(":commit")
	labl := r.FormValue(":label")

	// get the commit from the database
	commit, err := database.GetCommitHash(hash, repo.ID)
	if err != nil {
		return err
	}

	// get the builds from the database. a commit can have
	// multiple sub-builds (or matrix builds)
	builds, err := database.ListBuilds(commit.ID)
	if err != nil {
		return err
	}

	admin, err := database.IsRepoAdmin(u, repo)
	if err != nil {
		return err
	}

	data := struct {
		User    *User
		Repo    *Repo
		Commit  *Commit
		Build   *Build
		Builds  []*Build
		Token   string
		IsAdmin bool
	}{u, repo, commit, builds[0], builds, "", admin}

	// get the specific build requested by the user. instead
	// of a database round trip, we can just loop through the
	// list and extract the requested build.
	for _, b := range builds {
		if b.Slug == labl {
			data.Build = b
			break
		}
	}

	// generate a token to connect with the websocket
	// handler and stream output, if the build is running.
	data.Token = channel.Token(fmt.Sprintf(
		"%s/%s/%s/commit/%s/builds/%s", repo.Host, repo.Owner, repo.Name, commit.Hash, builds[0].Slug))

	// render the repository template.
	return RenderTemplate(w, "repo_commit.html", &data)
}

type CommitRebuildHandler struct {
	queue *queue.Queue
}

func NewCommitRebuildHandler(queue *queue.Queue) *CommitRebuildHandler {
	return &CommitRebuildHandler{
		queue: queue,
	}
}

// CommitRebuild re-queues a previously built commit. It finds the existing
// commit and build and injects them back into the queue.  If the commit
// doesn't exist or has no builds, or if a build label has been passed but
// can't be located, it prints an error. If the .drone.yml doesn't exist in
// the repo or can't be parsed, it saves a failed build and prints an error.
// Otherwise, it adds the build/commit to the queue and redirects back to the
// commit page.
func (h *CommitRebuildHandler) CommitRebuild(w http.ResponseWriter, r *http.Request, u *User, repo *Repo) error {
	hash := r.FormValue(":commit")
	labl := r.FormValue(":label")
	host := r.FormValue(":host")

	// get the commit from the database
	commit, err := database.GetCommitHash(hash, repo.ID)
	if err != nil {
		return err
	}

	// get the builds from the database. a commit can have
	// multiple sub-builds (or matrix builds)
	builds, err := database.ListBuilds(commit.ID)
	if err != nil {
		return err
	}

	build := builds[0]

	if labl != "" {
		// get the specific build requested by the user. instead
		// of a database round trip, we can just loop through the
		// list and extract the requested build.
		build = nil
		for _, b := range builds {
			if b.Slug == labl {
				build = b
				break
			}
		}
	}

	if build == nil {
		return fmt.Errorf("Could not find build: %s", labl)
	}

	buildscript, err := fetchBuildScript(repo, commit, u.GithubToken)
	if err != nil {
		return err
	}

	h.queue.Add(&queue.BuildTask{Repo: repo, Commit: commit, Build: build, Script: buildscript})

	if labl != "" {
		http.Redirect(w, r, fmt.Sprintf("/%s/%s/%s/commit/%s/build/%s", host, repo.Owner, repo.Name, hash, labl), http.StatusSeeOther)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/%s/%s/%s/commit/%s", host, repo.Owner, repo.Name, hash), http.StatusSeeOther)
	}

	return nil
}
