package handler

import (
	"fmt"
	"net/http"

	"github.com/drone/drone/pkg/channel"
	"github.com/drone/drone/pkg/database"
	. "github.com/drone/drone/pkg/model"
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

	data := struct {
		User   *User
		Repo   *Repo
		Commit *Commit
		Build  *Build
		Builds []*Build
		Token  string
	}{u, repo, commit, builds[0], builds, ""}

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
