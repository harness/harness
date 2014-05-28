package handler

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/drone/drone/pkg/database"
	. "github.com/drone/drone/pkg/model"
	"github.com/drone/drone/pkg/queue"
	"github.com/drone/go-github/github"
)

type GithubHandler struct {
	queue *queue.Queue
}

func NewGithubHandler(queue *queue.Queue) *GithubHandler {
	return &GithubHandler{
		queue: queue,
	}
}

// Processes a generic POST-RECEIVE GitHub hook and
// attempts to trigger a build.
func (h *GithubHandler) Hook(w http.ResponseWriter, r *http.Request) error {
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
		println("could not parse hook:", err.Error())
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

	// Get the user that owns the repository
	user, err := database.GetUser(repo.UserID)
	if err != nil {
		return RenderText(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	}

	// Verify that the commit doesn't already exist.
	// We should never build the same commit twice.
	_, err = database.GetCommitBranchHash(hook.Branch(), hook.Head.Id, repo.ID)
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

	// get the github settings from the database
	settings := database.SettingsMust()

	// get the drone.yml file from GitHub
	client := github.New(user.GithubToken)
	client.ApiUrl = settings.GitHubApiUrl

	content, err := client.Contents.FindRef(repo.Owner, repo.Name, ".drone.yml", commit.Hash)
	if err != nil {
		msg := "No .drone.yml was found in this repository.  You need to add one.\n"
		if err := saveFailedBuild(commit, msg); err != nil {
			return RenderText(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return RenderText(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	}

	// decode the content.  Note: Not sure this will ever happen...it basically means a GitHub API issue
	buildscript, err := content.DecodeContent()
	if err != nil {
		return RenderText(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
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
	build.BuildScript = string(buildscript)
	if err := database.SaveBuild(build); err != nil {
		return RenderText(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	// notify websocket that a new build is pending
	//realtime.CommitPending(repo.UserID, repo.TeamID, repo.ID, commit.ID, repo.Private)
	//realtime.BuildPending(repo.UserID, repo.TeamID, repo.ID, commit.ID, build.ID, repo.Private)

	h.queue.Add(&queue.BuildTask{Repo: repo, Commit: commit, Build: build}) //Push(repo, commit, build)

	// OK!
	return RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
}

func (h *GithubHandler) PullRequestHook(w http.ResponseWriter, r *http.Request) {
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

	// Get the user that owns the repository
	user, err := database.GetUser(repo.UserID)
	if err != nil {
		RenderText(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
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
	// label := p.PullRequest.Head.Labe

	// get the github settings from the database
	settings := database.SettingsMust()

	// get the drone.yml file from GitHub
	client := github.New(user.GithubToken)
	client.ApiUrl = settings.GitHubApiUrl

	content, err := client.Contents.FindRef(repo.Owner, repo.Name, ".drone.yml", commit.Hash) // TODO should this really be the hash??
	if err != nil {
		println(err.Error())
		RenderText(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	// decode the content
	buildscript, err := content.DecodeContent()
	if err != nil {
		RenderText(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
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
	build.BuildScript = string(buildscript)
	if err := database.SaveBuild(build); err != nil {
		RenderText(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// notify websocket that a new build is pending
	// TODO we should, for consistency, just put this inside Queue.Add()
	h.queue.Add(&queue.BuildTask{Repo: repo, Commit: commit, Build: build})

	// OK!
	RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
}
