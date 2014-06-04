package handler

import (
	"net/http"
)

func parseRepo(r *http.Request) (host string, owner string, name string) {
	host = r.FormValue(":host")
	owner = r.FormValue(":owner")
	name = r.FormValue(":name")
	return
}

func parseBranch(r *http.Request) (branch string) {
	return r.FormValue(":branch")
}

func parseCommit(r *http.Request) (commit string) {
	return r.FormValue(":commit")
}
