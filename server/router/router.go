package router

import (
	"regexp"

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
	mux.Get("/api/hook/:host/:token", handler.PostHook)
	mux.Put("/api/hook/:host/:token", handler.PostHook)
	mux.Post("/api/hook/:host/:token", handler.PostHook)

	// these routes are here for backward compatibility
	// to help people troubleshoot why their upgrade isn't
	// working correctly. remove at some point
	mux.Get("/api/hook/:host", handler.PostHook)
	mux.Put("/api/hook/:host", handler.PostHook)
	mux.Post("/api/hook/:host", handler.PostHook)
	////

	streams := web.New()
	streams.Get("/api/stream/stdout/:id", handler.WsConsole)
	streams.Get("/api/stream/user", handler.WsUser)
	mux.Handle("/api/stream/*", streams)

	repos := web.New()
	repos.Use(middleware.SetRepo)
	repos.Use(middleware.RequireRepoRead)
	repos.Use(middleware.RequireRepoAdmin)
	repos.Get(regexp.MustCompile(`^\/api\/repos\/(?P<host>(.*))\/(?P<owner>(.*))\/(?P<name>(.*))\/branches\/(?P<branch>(.*))\/commits\/(?P<commit>(.*))\/console$`), handler.GetOutput)
	repos.Get(regexp.MustCompile(`^\/api\/repos\/(?P<host>(.*))\/(?P<owner>(.*))\/(?P<name>(.*))\/branches\/(?P<branch>(.*))\/commits\/(?P<commit>(.*))$`), handler.GetCommit)
	repos.Post(regexp.MustCompile(`^\/api\/repos\/(?P<host>(.*))\/(?P<owner>(.*))\/(?P<name>(.*))\/branches\/(?P<branch>(.*))\/commits\/(?P<commit>(.*))$`), handler.PostCommit)
	repos.Get("/api/repos/:host/:owner/:name/commits", handler.GetCommitList)
	repos.Get("/api/repos/:host/:owner/:name", handler.GetRepo)
	repos.Put("/api/repos/:host/:owner/:name", handler.PutRepo)
	repos.Post("/api/repos/:host/:owner/:name", handler.PostRepo)
	repos.Delete("/api/repos/:host/:owner/:name", handler.DelRepo)
	repos.Post("/api/repos/:host/:owner/:name/deactivate", handler.DeactivateRepo)
	mux.Handle("/api/repos/:host/:owner/:name", repos)
	mux.Handle("/api/repos/:host/:owner/:name/*", repos)

	users := web.New()
	users.Use(middleware.RequireUserAdmin)
	users.Get("/api/users/:host/:login", handler.GetUser)
	users.Post("/api/users/:host/:login", handler.PostUser)
	users.Delete("/api/users/:host/:login", handler.DelUser)
	users.Get("/api/users", handler.GetUserList)
	mux.Handle("/api/users", users)
	mux.Handle("/api/users/*", users)

	user := web.New()
	user.Use(middleware.RequireUser)
	user.Get("/api/user/feed", handler.GetUserFeed)
	user.Get("/api/user/activity", handler.GetUserActivity)
	user.Get("/api/user/repos", handler.GetUserRepos)
	user.Post("/api/user/sync", handler.PostUserSync)
	user.Get("/api/user", handler.GetUserCurrent)
	user.Put("/api/user", handler.PutUser)
	mux.Handle("/api/user", user)
	mux.Handle("/api/user/*", user)

	work := web.New()
	work.Use(middleware.RequireUserAdmin)
	work.Get("/api/work/started", handler.GetWorkStarted)
	work.Get("/api/work/pending", handler.GetWorkPending)
	work.Get("/api/work/assignments", handler.GetWorkAssigned)
	work.Get("/api/workers", handler.GetWorkers)
	work.Post("/api/workers", handler.PostWorker)
	work.Delete("/api/workers", handler.DelWorker)
	mux.Handle("/api/work", work)
	mux.Handle("/api/work/*", work)

	return mux
}
