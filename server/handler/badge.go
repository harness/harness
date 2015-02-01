package handler

import (
	"encoding/xml"
	"net/http"

	"github.com/drone/drone/server/datastore"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/model"
	"github.com/goji/context"
	"github.com/zenazn/goji/web"
)

// badges that indicate the current build status for a repository
// and branch combination.
type badge struct {
	success, failure, started, err, none []byte
}

var defaultBadge = badge{
	[]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="91" height="18"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#fff" stop-opacity=".7"/><stop offset=".1" stop-color="#aaa" stop-opacity=".1"/><stop offset=".9" stop-opacity=".3"/><stop offset="1" stop-opacity=".5"/></linearGradient><rect rx="4" width="91" height="18" fill="#555"/><rect rx="4" x="37" width="54" height="18" fill="#4c1"/><path fill="#4c1" d="M37 0h4v18h-4z"/><rect rx="4" width="91" height="18" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="13" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="12">build</text><text x="63" y="13" fill="#010101" fill-opacity=".3">success</text><text x="63" y="12">success</text></g></svg>`),
	[]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="83" height="18"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#fff" stop-opacity=".7"/><stop offset=".1" stop-color="#aaa" stop-opacity=".1"/><stop offset=".9" stop-opacity=".3"/><stop offset="1" stop-opacity=".5"/></linearGradient><rect rx="4" width="83" height="18" fill="#555"/><rect rx="4" x="37" width="46" height="18" fill="#e05d44"/><path fill="#e05d44" d="M37 0h4v18h-4z"/><rect rx="4" width="83" height="18" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="13" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="12">build</text><text x="59" y="13" fill="#010101" fill-opacity=".3">failure</text><text x="59" y="12">failure</text></g></svg>`),
	[]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="87" height="18"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#fff" stop-opacity=".7"/><stop offset=".1" stop-color="#aaa" stop-opacity=".1"/><stop offset=".9" stop-opacity=".3"/><stop offset="1" stop-opacity=".5"/></linearGradient><rect rx="4" width="87" height="18" fill="#555"/><rect rx="4" x="37" width="50" height="18" fill="#dfb317"/><path fill="#dfb317" d="M37 0h4v18h-4z"/><rect rx="4" width="87" height="18" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="13" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="12">build</text><text x="61" y="13" fill="#010101" fill-opacity=".3">started</text><text x="61" y="12">started</text></g></svg>`),
	[]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="76" height="18"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#fff" stop-opacity=".7"/><stop offset=".1" stop-color="#aaa" stop-opacity=".1"/><stop offset=".9" stop-opacity=".3"/><stop offset="1" stop-opacity=".5"/></linearGradient><rect rx="4" width="76" height="18" fill="#555"/><rect rx="4" x="37" width="39" height="18" fill="#9f9f9f"/><path fill="#9f9f9f" d="M37 0h4v18h-4z"/><rect rx="4" width="76" height="18" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="13" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="12">build</text><text x="55.5" y="13" fill="#010101" fill-opacity=".3">error</text><text x="55.5" y="12">error</text></g></svg>`),
	[]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="75" height="18"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#fff" stop-opacity=".7"/><stop offset=".1" stop-color="#aaa" stop-opacity=".1"/><stop offset=".9" stop-opacity=".3"/><stop offset="1" stop-opacity=".5"/></linearGradient><rect rx="4" width="75" height="18" fill="#555"/><rect rx="4" x="37" width="38" height="18" fill="#9f9f9f"/><path fill="#9f9f9f" d="M37 0h4v18h-4z"/><rect rx="4" width="75" height="18" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="13" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="12">build</text><text x="55" y="13" fill="#010101" fill-opacity=".3">none</text><text x="55" y="12">none</text></g></svg>`),
}

var badgeStyles = map[string]badge{
	"flat": badge{
		[]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="91" height="20"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><rect rx="3" width="91" height="20" fill="#555"/><rect rx="3" x="37" width="54" height="20" fill="#4c1"/><path fill="#4c1" d="M37 0h4v20h-4z"/><rect rx="3" width="91" height="20" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="15" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="14">build</text><text x="63" y="15" fill="#010101" fill-opacity=".3">success</text><text x="63" y="14">success</text></g></svg>`),
		[]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="83" height="20"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><rect rx="3" width="83" height="20" fill="#555"/><rect rx="3" x="37" width="46" height="20" fill="#e05d44"/><path fill="#e05d44" d="M37 0h4v20h-4z"/><rect rx="3" width="83" height="20" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="15" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="14">build</text><text x="59" y="15" fill="#010101" fill-opacity=".3">failure</text><text x="59" y="14">failure</text></g></svg>`),
		[]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="87" height="20"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><rect rx="3" width="87" height="20" fill="#555"/><rect rx="3" x="37" width="50" height="20" fill="#dfb317"/><path fill="#dfb317" d="M37 0h4v20h-4z"/><rect rx="3" width="87" height="20" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="15" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="14">build</text><text x="61" y="15" fill="#010101" fill-opacity=".3">started</text><text x="61" y="14">started</text></g></svg>`),
		[]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="76" height="20"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><rect rx="3" width="76" height="20" fill="#555"/><rect rx="3" x="37" width="39" height="20" fill="#9f9f9f"/><path fill="#9f9f9f" d="M37 0h4v20h-4z"/><rect rx="3" width="76" height="20" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="15" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="14">build</text><text x="55.5" y="15" fill="#010101" fill-opacity=".3">error</text><text x="55.5" y="14">error</text></g></svg>`),
		[]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="75" height="20"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><rect rx="3" width="75" height="20" fill="#555"/><rect rx="3" x="37" width="38" height="20" fill="#9f9f9f"/><path fill="#9f9f9f" d="M37 0h4v20h-4z"/><rect rx="3" width="75" height="20" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="15" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="14">build</text><text x="55" y="15" fill="#010101" fill-opacity=".3">none</text><text x="55" y="14">none</text></g></svg>`),
	},
	"flat-square": badge{
		[]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="91" height="20"><path fill="#555" d="M0 0h91v20h-91z"/><path fill="#4c1" d="M37 0h54v20h-54zM37 0h4v20h-4z"/><path fill-opacity=".1" d="M0 0h91v20h-91z"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="14">build</text><text x="63" y="14">success</text></g></svg>`),
		[]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="83" height="20"><path fill="#555" d="M0 0h83v20h-83z"/><path fill="#e05d44" d="M37 0h46v20h-46zM37 0h4v20h-4z"/><path fill-opacity=".1" d="M0 0h83v20h-83z"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="14">build</text><text x="59" y="14">failure</text></g></svg>`),
		[]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="87" height="20"><path fill="#555" d="M0 0h87v20h-87z"/><path fill="#dfb317" d="M37 0h50v20h-50zM37 0h4v20h-4z"/><path fill-opacity=".1" d="M0 0h87v20h-87z"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="14">build</text><text x="61" y="14">started</text></g></svg>`),
		[]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="76" height="20"><path fill="#555" d="M0 0h76v20h-76z"/><path fill="#9f9f9f" d="M37 0h39v20h-39zM37 0h4v20h-4z"/><path fill-opacity=".1" d="M0 0h76v20h-76z"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="14">build</text><text x="55.5" y="14">error</text></g></svg>`),
		[]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="75" height="20"><path fill="#555" d="M0 0h75v20h-75z"/><path fill="#9f9f9f" d="M37 0h38v20h-38zM37 0h4v20h-4z"/><path fill-opacity=".1" d="M0 0h75v20h-75z"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="14">build</text><text x="55" y="14">none</text></g></svg>`),
	},
}

// GetBadge accepts a request to retrieve the named
// repo and branhes latest build details from the datastore
// and return an SVG badges representing the build results.
//
//     GET /api/badge/:host/:owner/:name/status.svg
//
func GetBadge(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)
	var (
		host   = c.URLParams["host"]
		owner  = c.URLParams["owner"]
		name   = c.URLParams["name"]
		branch = r.FormValue("branch")
		style  = r.FormValue("style")
	)

	// an SVG response is always served, even when error, so
	// we can go ahead and set the content type appropriately.
	w.Header().Set("Content-Type", "image/svg+xml")

	badge, ok := badgeStyles[style]
	if !ok {
		badge = defaultBadge
	}

	repo, err := datastore.GetRepoName(ctx, host, owner, name)
	if err != nil {
		w.Write(badge.none)
		return
	}
	if len(branch) == 0 {
		branch = model.DefaultBranch
	}
	commit, _ := datastore.GetCommitLast(ctx, repo, branch)

	// if no commit was found then display
	// the 'none' badge, instead of throwing
	// an error response
	if commit == nil {
		w.Write(badge.none)
		return
	}

	switch commit.Status {
	case model.StatusSuccess:
		w.Write(badge.success)
	case model.StatusFailure:
		w.Write(badge.failure)
	case model.StatusError:
		w.Write(badge.err)
	case model.StatusEnqueue, model.StatusStarted:
		w.Write(badge.started)
	default:
		w.Write(badge.none)
	}
}

// GetCC accepts a request to retrieve the latest build
// status for the given repository from the datastore and
// in CCTray XML format.
//
//     GET /api/badge/:host/:owner/:name/cc.xml
//
func GetCC(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)
	var (
		host  = c.URLParams["host"]
		owner = c.URLParams["owner"]
		name  = c.URLParams["name"]
	)

	w.Header().Set("Content-Type", "application/xml")

	repo, err := datastore.GetRepoName(ctx, host, owner, name)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	commits, err := datastore.GetCommitList(ctx, repo)
	if err != nil || len(commits) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var link = httputil.GetURL(r) + "/" + repo.Host + "/" + repo.Owner + "/" + repo.Name
	var cc = model.NewCC(repo, commits[0], link)
	xml.NewEncoder(w).Encode(cc)
}
