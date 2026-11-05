package api

import (
	"crypto/subtle"
	"net/http"
	"net/url"
	"strings"

	"github.com/drone/drone/core"
	"github.com/drone/drone/handler/api/request"
	"github.com/go-chi/cors"
)

// browserSecurity runs after authentication. Cookies are ambient browser
// credentials; an explicitly supplied, authenticated API token is not.
func browserSecurity(system *core.System) func(http.Handler) http.Handler {
	var publicOrigin string
	if system != nil {
		publicOrigin = system.Proto + "://" + system.Host
	}
	trusted := func(origin string) bool { return sameOrigin(publicOrigin, origin, false) }
	c := cors.New(cors.Options{
		AllowOriginFunc: func(r *http.Request, origin string) bool {
			if len(r.Header.Values("Origin")) != 1 {
				return false
			}
			if authenticatedAPIToken(r) {
				return true
			}
			if origin != "null" && parsedOrigin(origin, false) == nil {
				return false
			}
			if trusted(origin) {
				return true
			}
			// Preflights cannot carry the token itself. Permit the Authorization
			// header here, then require a valid token on the actual request.
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				for _, header := range strings.Split(r.Header.Get("Access-Control-Request-Headers"), ",") {
					if strings.EqualFold(strings.TrimSpace(header), "Authorization") {
						return true
					}
				}
			}
			return false
		},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	})
	return func(next http.Handler) http.Handler {
		guard := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
			default:
				if !authenticatedAPIToken(r) && !trustedBrowserRequest(r, trusted, publicOrigin) {
					http.Error(w, "cross-origin request forbidden", http.StatusForbidden)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
		handler := c.Handler(guard)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Vary", "Authorization")
			handler.ServeHTTP(w, r)
		})
	}
}

func authenticatedAPIToken(r *http.Request) bool {
	// Match session.Get's precedence and support legacy access_token clients.
	token := r.Header.Get("Authorization")
	if token == "" {
		token = r.FormValue("access_token")
	}
	token = strings.TrimPrefix(token, "Bearer ")
	user, ok := request.UserFrom(r.Context())
	// Do not exempt a request merely because it supplies a token:
	// it must match the authenticated user's API token, never a session cookie.
	return ok && user != nil && token != "" && subtle.ConstantTimeCompare([]byte(token), []byte(user.Hash)) == 1
}

func trustedBrowserRequest(r *http.Request, trusted func(string) bool, publicOrigin string) bool {
	if origins := r.Header.Values("Origin"); len(origins) != 0 {
		return len(origins) == 1 && trusted(origins[0])
	}
	// Some older clients omit Origin. Use available browser provenance without
	// breaking non-browser clients that send none of these headers.
	if sites := r.Header.Values("Sec-Fetch-Site"); len(sites) != 0 {
		if len(sites) != 1 || (sites[0] != "same-origin" && sites[0] != "none") {
			return false
		}
	}
	if referers := r.Header.Values("Referer"); len(referers) != 0 {
		return len(referers) == 1 && sameOrigin(publicOrigin, referers[0], true)
	}
	return true
}

// Compare URL origins, allowing equivalent host casing and default ports.
// Always use the configured user-facing URL, which survives reverse proxies;
// client-controlled Host and forwarding headers must not expand this trust.
func sameOrigin(public, candidate string, referer bool) bool {
	a, b := parsedOrigin(public, false), parsedOrigin(candidate, referer)
	return a != nil && b != nil && a.Scheme == b.Scheme && strings.EqualFold(a.Hostname(), b.Hostname()) && a.Port() == b.Port()
}

func parsedOrigin(raw string, referer bool) *url.URL {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Opaque != "" || u.Fragment != "" {
		return nil
	}
	if !referer && (u.Path != "" || u.RawQuery != "" || u.ForceQuery) {
		return nil
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil
	}
	if (u.Scheme == "https" && u.Port() == "443") || (u.Scheme == "http" && u.Port() == "80") {
		u.Host = strings.TrimSuffix(u.Host, ":"+u.Port())
	}
	return u
}
