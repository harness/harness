package router

import (
	"github.com/drone/drone/server/handler"
	"github.com/drone/drone/server/middleware"

	"github.com/zenazn/goji/web"
)

func New() *web.Mux {
	mux := web.New()

	mux.Get("/api/logins", handler.GetLoginList)
	mux.Get("/api/auth/:host", handler.GetLogin)
	mux.Post("/api/auth/:host", handler.GetLogin)
	mux.Get("/api/badge/:host/:owner/:name/status.svg", handler.GetBadge)
	mux.Get("/api/badge/:host/:owner/:name/cc.xml", handler.GetCC)
	mux.Get("/api/hook/:host", handler.PostHook)
	mux.Put("/api/hook/:host", handler.PostHook)
	mux.Post("/api/hook/:host", handler.PostHook)

	streams := web.New()
	streams.Get("/api/stream/stdout/:id", handler.WsConsole)
	streams.Get("/api/stream/user", handler.WsUser)
	mux.Handle("/api/stream/*", streams)

	repos := web.New()
	repos.Use(middleware.SetRepo)
	repos.Use(middleware.RequireRepoRead)
	repos.Use(middleware.RequireRepoAdmin)
	repos.Get("/api/repos/:host/:owner/:name/branches/:branch/commits/:commit/console", handler.GetOutput)
	repos.Get("/api/repos/:host/:owner/:name/branches/:branch/commits/:commit", handler.GetCommit)
	repos.Post("/api/repos/:host/:owner/:name/branches/:branch/commits/:commit", handler.PostCommit)
	repos.Get("/api/repos/:host/:owner/:name/commits", handler.GetCommitList)
	repos.Get("/api/repos/:host/:owner/:name", handler.GetRepo)
	repos.Put("/api/repos/:host/:owner/:name", handler.PutRepo)
	repos.Post("/api/repos/:host/:owner/:name", handler.PostRepo)
	repos.Delete("/api/repos/:host/:owner/:name", handler.DelRepo)
	mux.Handle("/api/repos/:host/:owner/:name*", repos)

	users := web.New()
	users.Use(middleware.RequireUserAdmin)
	users.Get("/api/users/:host/:login", handler.GetUser)
	users.Post("/api/users/:host/:login", handler.PostUser)
	users.Delete("/api/users/:host/:login", handler.DelUser)
	users.Get("/api/users", handler.GetUserList)
	mux.Handle("/api/users*", users)

	user := web.New()
	user.Use(middleware.RequireUser)
	user.Get("/api/user/feed", handler.GetUserFeed)
	user.Get("/api/user/repos", handler.GetUserRepos)
	user.Get("/api/user", handler.GetUserCurrent)
	user.Put("/api/user", handler.PutUser)
	mux.Handle("/api/user*", user)

	work := web.New()
	work.Use(middleware.RequireUserAdmin)
	work.Get("/api/work/started", handler.GetWorkStarted)
	work.Get("/api/work/pending", handler.GetWorkPending)
	work.Get("/api/work/assignments", handler.GetWorkAssigned)
	work.Get("/api/workers", handler.GetWorkers)
	work.Post("/api/workers", handler.PostWorker)
	work.Delete("/api/workers", handler.DelWorker)
	mux.Handle("/api/work*", work)

	return mux
}
