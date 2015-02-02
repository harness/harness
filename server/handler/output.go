package handler

import (
	"io"
	"net/http"
	"path/filepath"

	"github.com/drone/drone/server/blobstore"
	"github.com/goji/context"
	"github.com/zenazn/goji/web"
)

// GetOutput gets the commit's stdout.
//
//     GET /api/repos/:host/:owner/:name/branches/:branch/commits/:commit/console
//
func GetOutput(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)
	var (
		host   = c.URLParams["host"]
		owner  = c.URLParams["owner"]
		name   = c.URLParams["name"]
		branch = c.URLParams["branch"]
		hash   = c.URLParams["commit"]
	)

	w.Header().Set("Content-Type", "text/plain")

	path := filepath.Join(host, owner, name, branch, hash)
	rc, err := blobstore.GetReader(ctx, path)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	defer rc.Close()
	io.Copy(w, rc)
}
