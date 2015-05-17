package server

import (
	"github.com/gin-gonic/gin"

	"github.com/drone/drone/pkg/ccmenu"
	common "github.com/drone/drone/pkg/types"
)

var (
	badgeSuccess = []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="91" height="20"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><rect rx="3" width="91" height="20" fill="#555"/><rect rx="3" x="37" width="54" height="20" fill="#4c1"/><path fill="#4c1" d="M37 0h4v20h-4z"/><rect rx="3" width="91" height="20" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="15" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="14">build</text><text x="63" y="15" fill="#010101" fill-opacity=".3">success</text><text x="63" y="14">success</text></g></svg>`)
	badgeFailure = []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="83" height="20"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><rect rx="3" width="83" height="20" fill="#555"/><rect rx="3" x="37" width="46" height="20" fill="#e05d44"/><path fill="#e05d44" d="M37 0h4v20h-4z"/><rect rx="3" width="83" height="20" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="15" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="14">build</text><text x="59" y="15" fill="#010101" fill-opacity=".3">failure</text><text x="59" y="14">failure</text></g></svg>`)
	badgeStarted = []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="87" height="20"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><rect rx="3" width="87" height="20" fill="#555"/><rect rx="3" x="37" width="50" height="20" fill="#dfb317"/><path fill="#dfb317" d="M37 0h4v20h-4z"/><rect rx="3" width="87" height="20" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="15" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="14">build</text><text x="61" y="15" fill="#010101" fill-opacity=".3">started</text><text x="61" y="14">started</text></g></svg>`)
	badgeError   = []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="76" height="20"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><rect rx="3" width="76" height="20" fill="#555"/><rect rx="3" x="37" width="39" height="20" fill="#9f9f9f"/><path fill="#9f9f9f" d="M37 0h4v20h-4z"/><rect rx="3" width="76" height="20" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="15" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="14">build</text><text x="55.5" y="15" fill="#010101" fill-opacity=".3">error</text><text x="55.5" y="14">error</text></g></svg>`)
	badgeNone    = []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="75" height="20"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><rect rx="3" width="75" height="20" fill="#555"/><rect rx="3" x="37" width="38" height="20" fill="#9f9f9f"/><path fill="#9f9f9f" d="M37 0h4v20h-4z"/><rect rx="3" width="75" height="20" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="15" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="14">build</text><text x="55" y="15" fill="#010101" fill-opacity=".3">none</text><text x="55" y="14">none</text></g></svg>`)
)

// GetBadge accepts a request to retrieve the named
// repo and branhes latest build details from the datastore
// and return an SVG badges representing the build results.
//
//     GET /api/badge/:owner/:name/status.svg
//
func GetBadge(c *gin.Context) {
	var repo = ToRepo(c)
	var store = ToDatastore(c)
	var branch = c.Request.FormValue("branch")
	if len(branch) == 0 {
		branch = repo.Branch
	}

	// an SVG response is always served, even when error, so
	// we can go ahead and set the content type appropriately.
	c.Writer.Header().Set("Content-Type", "image/svg+xml")

	// if no commit was found then display
	// the 'none' badge, instead of throwing
	// an error response
	commit, err := store.CommitLast(repo, branch)
	if err != nil {
		c.Writer.Write(badgeNone)
		return
	}

	switch commit.State {
	case common.StateSuccess:
		c.Writer.Write(badgeSuccess)
	case common.StateFailure:
		c.Writer.Write(badgeFailure)
	case common.StateError, common.StateKilled:
		c.Writer.Write(badgeError)
	case common.StatePending, common.StateRunning:
		c.Writer.Write(badgeStarted)
	default:
		c.Writer.Write(badgeNone)
	}
}

// GetCC accepts a request to retrieve the latest build
// status for the given repository from the datastore and
// in CCTray XML format.
//
//     GET /api/badge/:host/:owner/:name/cc.xml
//
// TODO(bradrydzewski) this will not return in-progress builds, which it should
func GetCC(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	list, err := store.CommitList(repo, 1, 0)
	if err != nil || len(list) == 0 {
		c.AbortWithStatus(404)
		return
	}

	cc := ccmenu.NewCC(repo, list[0])

	c.Writer.Header().Set("Content-Type", "application/xml")
	c.XML(200, cc)
}
