package handler

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/drone/drone/pkg/build/script"
	"github.com/drone/drone/pkg/database"
	. "github.com/drone/drone/pkg/model"
	"github.com/drone/drone/pkg/queue"
	"github.com/drone/go-github/github"
)

type HookHandler struct {
	queue *queue.Queue
}

func NewHookHandler(queue *queue.Queue) *HookHandler {
	return &HookHandler{
		queue: queue,
	}
}

// Processes a generic POST-RECEIVE hook and
// attempts to trigger a build.
func (h *HookHandler) Hook(w http.ResponseWriter, r *http.Request) error {
	// handle github ping
	if r.Header.Get("X-Github-Event") == "ping" {
		return RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
	}

	// if this is a pull request route
	// to a different handler
	if r.Header.Get("X-Github-Event") == "pull_request" {
		h.PullRequestHook(w, r)
		return nil
	}

	// get the payload of the message
	// this should contain a json representation of the
	// repository and commit details
	payload := r.FormValue("payload")

	// parse the github Hook payload
	hook, err := github.ParseHook([]byte(payload))
	if err != nil {
		println("could not parse hook")
		return RenderText(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	}

	// make sure this is being triggered because of a commit
	// and not something like a tag deletion or whatever
	if hook.IsTag() || hook.IsGithubPages() ||
		hook.IsHead() == false || hook.IsDeleted() {
		return RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
	}

	// get the repo from the URL
	repoId := r.FormValue("id")

	// get the repo from the database, return error if not found
	repo, err := database.GetRepoSlug(repoId)
	if err != nil {
		return RenderText(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}

	// Verify that the commit doesn't already exist.
	// We should never build the same commit twice.
	_, err = database.GetCommitHash(hook.Head.Id, repo.ID)
	if err != nil && err != sql.ErrNoRows {
		println("commit already exists")
		return RenderText(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
	}

	// we really only need:
	//  * repo owner
	//  * repo name
	//  * repo host (github)
	//  * commit hash
	//  * commit timestamp
	//  * commit branch
	//  * commit message
	//  * commit author
	//  * pull request

	// once we have this data we could just send directly to the queue
	// and let it handle everything else

	commit := &Commit{}
	commit.RepoID = repo.ID
	commit.Branch = hook.Branch()
	commit.Hash = hook.Head.Id
	commit.Status = "Pending"
	commit.Created = time.Now().UTC()

	// extract the author and message from the commit
	// this is kind of experimental, since I don't know
	// what I'm doing here.
	if hook.Head != nil && hook.Head.Author != nil {
		commit.Message = hook.Head.Message
		commit.Timestamp = hook.Head.Timestamp
		commit.SetAuthor(hook.Head.Author.Email)
	} else if hook.Commits != nil && len(hook.Commits) > 0 && hook.Commits[0].Author != nil {
		commit.Message = hook.Commits[0].Message
		commit.Timestamp = hook.Commits[0].Timestamp
		commit.SetAuthor(hook.Commits[0].Author.Email)
	}

	buildscript, err := fetchBuildScript(repo, commit)
	if err != nil {
		println(err.Error())
		return RenderText(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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
	if err := database.SaveBuild(build); err != nil {
		return RenderText(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	// notify websocket that a new build is pending
	//realtime.CommitPending(repo.UserID, repo.TeamID, repo.ID, commit.ID, repo.Private)
	//realtime.BuildPending(repo.UserID, repo.TeamID, repo.ID, commit.ID, build.ID, repo.Private)

	h.queue.Add(&queue.BuildTask{Repo: repo, Commit: commit, Build: build, Script: buildscript}) //Push(repo, commit, build, buildscript)

	// OK!
	return RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
}

func (h *HookHandler) PullRequestHook(w http.ResponseWriter, r *http.Request) {
	// get the payload of the message
	// this should contain a json representation of the
	// repository and commit details
	payload := r.FormValue("payload")

	println("GOT PR HOOK")
	println(payload)

	hook, err := github.ParsePullRequestHook([]byte(payload))
	if err != nil {
		RenderText(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	// ignore these
	if hook.Action != "opened" && hook.Action != "synchronize" {
		RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
		return
	}

	// get the repo from the URL
	repoId := r.FormValue("id")

	// get the repo from the database, return error if not found
	repo, err := database.GetRepoSlug(repoId)
	if err != nil {
		RenderText(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	// Verify that the commit doesn't already exist.
	// We should enver build the same commit twice.
	_, err = database.GetCommitHash(hook.PullRequest.Head.Sha, repo.ID)
	if err != nil && err != sql.ErrNoRows {
		RenderText(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}

	///////////////////////////////////////////////////////

	commit := &Commit{}
	commit.RepoID = repo.ID
	commit.Branch = hook.PullRequest.Head.Ref
	commit.Hash = hook.PullRequest.Head.Sha
	commit.Status = "Pending"
	commit.Created = time.Now().UTC()
	commit.Gravatar = hook.PullRequest.User.GravatarId
	commit.Author = hook.PullRequest.User.Login
	commit.PullRequest = strconv.Itoa(hook.Number)
	commit.Message = hook.PullRequest.Title

	buildscript, err := fetchBuildScript(repo, commit)
	if err != nil {
		println(err.Error())
		RenderText(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// save the commit to the database
	if err := database.SaveCommit(commit); err != nil {
		RenderText(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// save the build to the database
	build := &Build{}
	build.Slug = "1" // TODO
	build.CommitID = commit.ID
	build.Created = time.Now().UTC()
	build.Status = "Pending"
	if err := database.SaveBuild(build); err != nil {
		RenderText(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// notify websocket that a new build is pending
	// TODO we should, for consistency, just put this inside Queue.Add()
	h.queue.Add(&queue.BuildTask{Repo: repo, Commit: commit, Build: build, Script: buildscript})

	// OK!
	RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
}

// fetchBuildScript retrieves a build script from GitHub. If the .drone.yml
// file is missing or incorrectly formatted, it returns an error.  Otherwise,
// it returns a Build suitable for adding to the queue
func fetchBuildScript(repo *Repo, commit *Commit) (*script.Build, error) {
	// get the github settings from the database
	settings := database.SettingsMust()

	// Get the user that owns the repository
	user, err := database.GetUser(repo.UserID)
	if err != nil {
		return nil, err
	}

	// get the drone.yml file from GitHub
	client := github.New(user.GithubToken)
	client.ApiUrl = settings.GitHubApiUrl

	content, err := client.Contents.FindRef(repo.Owner, repo.Name, ".drone.yml", commit.Hash)
	if err != nil {
		msg := "No .drone.yml was found in this repository.  You need to add one.\n"
		if err := saveFailedBuild(commit, msg); err != nil {
			return nil, err
		}
		return nil, err
	}

	// decode the content.  Note: Not sure this will ever happen...it basically means a GitHub API issue
	raw, err := content.DecodeContent()
	if err != nil {
		msg := "Could not decode the yaml from GitHub.  Check that your .drone.yml is a valid yaml file.\n"
		if err := saveFailedBuild(commit, msg); err != nil {
			return nil, err
		}
		return nil, err
	}

	// parse the build script
	buildscript, err := script.ParseBuild(raw, repo.Params)
	if err != nil {
		msg := "Could not parse your .drone.yml file.  It needs to be a valid drone yaml file.\n\n" + err.Error() + "\n"
		if err := saveFailedBuild(commit, msg); err != nil {
			return nil, err
		}
		return nil, err
	}

	return buildscript, nil
}

// Helper method for saving a failed build or commit in the case where it never starts to build.
// This can happen if the yaml is bad or doesn't exist.
func saveFailedBuild(commit *Commit, msg string) error {

	// Set the commit to failed
	commit.Status = "Failure"
	commit.Created = time.Now().UTC()
	commit.Finished = commit.Created
	commit.Duration = 0
	if err := database.SaveCommit(commit); err != nil {
		return err
	}

	// save the build to the database
	build := &Build{}
	build.Slug = "1" // TODO: This should not be hardcoded
	build.CommitID = commit.ID
	build.Created = time.Now().UTC()
	build.Finished = build.Created
	commit.Duration = 0
	build.Status = "Failure"
	build.Stdout = msg
	if err := database.SaveBuild(build); err != nil {
		return err
	}

	// TODO: Should the status be Error instead of Failure?

	// TODO: Do we need to update the branch table too?

	return nil

}
