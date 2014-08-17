package handler

import (
	"encoding/xml"
	"net/http"
	"time"

	"github.com/drone/drone/server/database"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/model"
	"github.com/gorilla/pat"
)

// badges that indicate the current build status for a repository
// and branch combination.
var (
	badgeSuccess = []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="91" height="18"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#fff" stop-opacity=".7"/><stop offset=".1" stop-color="#aaa" stop-opacity=".1"/><stop offset=".9" stop-opacity=".3"/><stop offset="1" stop-opacity=".5"/></linearGradient><rect rx="4" width="91" height="18" fill="#555"/><rect rx="4" x="37" width="54" height="18" fill="#4c1"/><path fill="#4c1" d="M37 0h4v18h-4z"/><rect rx="4" width="91" height="18" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="13" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="12">build</text><text x="63" y="13" fill="#010101" fill-opacity=".3">success</text><text x="63" y="12">success</text></g></svg>`)
	badgeFailure = []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="83" height="18"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#fff" stop-opacity=".7"/><stop offset=".1" stop-color="#aaa" stop-opacity=".1"/><stop offset=".9" stop-opacity=".3"/><stop offset="1" stop-opacity=".5"/></linearGradient><rect rx="4" width="83" height="18" fill="#555"/><rect rx="4" x="37" width="46" height="18" fill="#e05d44"/><path fill="#e05d44" d="M37 0h4v18h-4z"/><rect rx="4" width="83" height="18" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="13" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="12">build</text><text x="59" y="13" fill="#010101" fill-opacity=".3">failure</text><text x="59" y="12">failure</text></g></svg>`)
	badgeStarted = []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="87" height="18"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#fff" stop-opacity=".7"/><stop offset=".1" stop-color="#aaa" stop-opacity=".1"/><stop offset=".9" stop-opacity=".3"/><stop offset="1" stop-opacity=".5"/></linearGradient><rect rx="4" width="87" height="18" fill="#555"/><rect rx="4" x="37" width="50" height="18" fill="#dfb317"/><path fill="#dfb317" d="M37 0h4v18h-4z"/><rect rx="4" width="87" height="18" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="13" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="12">build</text><text x="61" y="13" fill="#010101" fill-opacity=".3">started</text><text x="61" y="12">started</text></g></svg>`)
	badgeError   = []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="76" height="18"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#fff" stop-opacity=".7"/><stop offset=".1" stop-color="#aaa" stop-opacity=".1"/><stop offset=".9" stop-opacity=".3"/><stop offset="1" stop-opacity=".5"/></linearGradient><rect rx="4" width="76" height="18" fill="#555"/><rect rx="4" x="37" width="39" height="18" fill="#9f9f9f"/><path fill="#9f9f9f" d="M37 0h4v18h-4z"/><rect rx="4" width="76" height="18" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="13" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="12">build</text><text x="55.5" y="13" fill="#010101" fill-opacity=".3">error</text><text x="55.5" y="12">error</text></g></svg>`)
	badgeNone    = []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="75" height="18"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#fff" stop-opacity=".7"/><stop offset=".1" stop-color="#aaa" stop-opacity=".1"/><stop offset=".9" stop-opacity=".3"/><stop offset="1" stop-opacity=".5"/></linearGradient><rect rx="4" width="75" height="18" fill="#555"/><rect rx="4" x="37" width="38" height="18" fill="#9f9f9f"/><path fill="#9f9f9f" d="M37 0h4v18h-4z"/><rect rx="4" width="75" height="18" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="13" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="12">build</text><text x="55" y="13" fill="#010101" fill-opacity=".3">none</text><text x="55" y="12">none</text></g></svg>`)
)

type BadgeHandler struct {
	commits database.CommitManager
	repos   database.RepoManager
}

func NewBadgeHandler(repos database.RepoManager, commits database.CommitManager) *BadgeHandler {
	return &BadgeHandler{commits, repos}
}

// GetStatus gets the build status badge.
// GET /v1/badge/:host/:owner/:name/status.svg
func (h *BadgeHandler) GetStatus(w http.ResponseWriter, r *http.Request) error {
	host, owner, name := parseRepo(r)
	branch := r.FormValue("branch")

	// github has insanely aggressive caching so we'll set almost
	// every parameter possible to try to prevent caching.
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Add("Cache-Control", "no-cache")
	w.Header().Add("Cache-Control", "no-store")
	w.Header().Add("Cache-Control", "max-age=0")
	w.Header().Add("Cache-Control", "must-revalidate")
	w.Header().Add("Cache-Control", "value")
	w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	w.Header().Set("Expires", "Thu, 01 Jan 1970 00:00:00 GMT")

	// get the repository from the database
	arepo, err := h.repos.FindName(host, owner, name)
	if err != nil {
		w.Write(badgeNone)
		return nil
	}

	// if no branch, use the default
	if len(branch) == 0 {
		branch = model.DefaultBranch
	}

	// get the latest commit
	c, _ := h.commits.FindLatest(arepo.Id, branch)

	// if no commit was found then display
	// the 'none' badge
	if c == nil {
		w.Write(badgeNone)
		return nil
	}

	// determine which badge to load
	switch c.Status {
	case model.StatusSuccess:
		w.Write(badgeSuccess)
	case model.StatusFailure:
		w.Write(badgeFailure)
	case model.StatusError:
		w.Write(badgeError)
	case model.StatusEnqueue, model.StatusStarted:
		w.Write(badgeStarted)
	default:
		w.Write(badgeNone)
	}

	return nil
}

// GetCoverage gets the build status badge.
// GET /v1/badges/:host/:owner/:name/coverage.svg
func (h *BadgeHandler) GetCoverage(w http.ResponseWriter, r *http.Request) error {
	return notImplemented{}
}

func (h *BadgeHandler) GetCC(w http.ResponseWriter, r *http.Request) error {
	host, owner, name := parseRepo(r)

	// get the repository from the database
	repo, err := h.repos.FindName(host, owner, name)
	if err != nil {
		return notFound{err}
	}

	// get the latest commits for the repo
	commits, err := h.commits.List(repo.Id)
	if err != nil || len(commits) == 0 {
		return notFound{}
	}
	commit := commits[0]

	// generate the URL for the repository
	url := httputil.GetURL(r) + "/" + repo.Host + "/" + repo.Owner + "/" + repo.Name
	proj := model.NewCC(repo, commit, url)
	return xml.NewEncoder(w).Encode(proj)
}

func (h *BadgeHandler) Register(r *pat.Router) {
	r.Get("/v1/badge/{host}/{owner}/{name}/coverage.svg", errorHandler(h.GetCoverage))
	r.Get("/v1/badge/{host}/{owner}/{name}/status.svg", errorHandler(h.GetStatus))
	r.Get("/v1/badge/{host}/{owner}/{name}/cc.xml", errorHandler(h.GetCC))
}
