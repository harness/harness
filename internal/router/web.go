package router

import (
	"net/http"

	"github.com/harness/gitness/internal/api/middleware/encode"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/web"
	"github.com/swaggest/swgui/v3emb"
	"github.com/unrolled/secure"

	"github.com/go-chi/chi"
)

/*
 * Mounts the WEB Router under mountPath.
 * The handler is wrapped within a layer that handles encoding Paths.
 */
func newWebHandler(
	mountPath string,
	systemStore store.SystemStore) (http.Handler, error) {

	config := systemStore.Config(nocontext)

	// Use go-chi router for inner routing (restricted to mountPath!)
	r := chi.NewRouter()
	r.Route(mountPath, func(r chi.Router) {

		// create middleware to enforce security best practices for
		// the user interface. note that theis middleware is only used
		// when serving the user interface (not found handler, below).
		sec := secure.New(
			secure.Options{
				AllowedHosts:          config.Secure.AllowedHosts,
				HostsProxyHeaders:     config.Secure.HostsProxyHeaders,
				SSLRedirect:           config.Secure.SSLRedirect,
				SSLTemporaryRedirect:  config.Secure.SSLTemporaryRedirect,
				SSLHost:               config.Secure.SSLHost,
				SSLProxyHeaders:       config.Secure.SSLProxyHeaders,
				STSSeconds:            config.Secure.STSSeconds,
				STSIncludeSubdomains:  config.Secure.STSIncludeSubdomains,
				STSPreload:            config.Secure.STSPreload,
				ForceSTSHeader:        config.Secure.ForceSTSHeader,
				FrameDeny:             config.Secure.FrameDeny,
				ContentTypeNosniff:    config.Secure.ContentTypeNosniff,
				BrowserXssFilter:      config.Secure.BrowserXSSFilter,
				ContentSecurityPolicy: config.Secure.ContentSecurityPolicy,
				ReferrerPolicy:        config.Secure.ReferrerPolicy,
			},
		)

		// openapi playground endpoints
		swagger := v3emb.NewHandler("API Definition", "/api/v1/swagger.yaml", "/swagger")
		r.With(sec.Handler).Handle("/swagger", swagger)
		r.With(sec.Handler).Handle("/swagger/*", swagger)

		// serve all other routes from the embedded filesystem,
		// which in turn serves the user interface.
		r.With(sec.Handler).NotFound(
			web.Handler(),
		)
	})

	// web doesn't have any prefixes for terminated paths
	return encode.TerminatedPathBefore([]string{""}, r.ServeHTTP), nil
}
