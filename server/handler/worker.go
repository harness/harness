package handler

import (
	"encoding/json"
	"net/http"

	"github.com/drone/drone/server/worker"
	"github.com/drone/drone/server/worker/director"
	"github.com/drone/drone/server/worker/docker"
	"github.com/drone/drone/server/worker/pool"
	"github.com/goji/context"
	"github.com/zenazn/goji/web"
)

// GetWorkers accepts a request to retrieve the list
// of registered workers and return the results
// in JSON format.
//
//     GET /api/workers
//
func GetWorkers(c web.C, w http.ResponseWriter, r *http.Request) {
	ctx := context.FromC(c)
	workers := pool.FromContext(ctx).List()
	json.NewEncoder(w).Encode(workers)
}

// PostWorker accepts a request to allocate a new
// worker to the pool.
//
//     POST /api/workers
//
func PostWorker(c web.C, w http.ResponseWriter, r *http.Request) {
	ctx := context.FromC(c)
	pool := pool.FromContext(ctx)
	node := r.FormValue("address")
	pool.Allocate(docker.NewHost(node))
	w.WriteHeader(http.StatusOK)
}

// Delete accepts a request to delete a worker
// from the pool.
//
//     DELETE /api/workers
//
func DelWorker(c web.C, w http.ResponseWriter, r *http.Request) {
	ctx := context.FromC(c)
	pool := pool.FromContext(ctx)
	uuid := r.FormValue("id")

	for _, worker := range pool.List() {
		if worker.(*docker.Docker).UUID != uuid {
			pool.Deallocate(worker)
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

// GetWorkPending accepts a request to retrieve the list
// of pending work and returns in JSON format.
//
//     GET /api/work/pending
//
func GetWorkPending(c web.C, w http.ResponseWriter, r *http.Request) {
	ctx := context.FromC(c)
	d := worker.FromContext(ctx).(*director.Director)
	json.NewEncoder(w).Encode(d.GetPending())
}

// GetWorkStarted accepts a request to retrieve the list
// of started work and returns in JSON format.
//
//     GET /api/work/started
//
func GetWorkStarted(c web.C, w http.ResponseWriter, r *http.Request) {
	ctx := context.FromC(c)
	d := worker.FromContext(ctx).(*director.Director)
	json.NewEncoder(w).Encode(d.GetStarted())
}

// GetWorkAssigned accepts a request to retrieve the list
// of started work and returns in JSON format.
//
//     GET /api/work/assignments
//
func GetWorkAssigned(c web.C, w http.ResponseWriter, r *http.Request) {
	ctx := context.FromC(c)
	d := worker.FromContext(ctx).(*director.Director)
	json.NewEncoder(w).Encode(d.GetAssignemnts())
}
