package location

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Resolve is a middleware function that resolves the hostname
// and scheme for the http.Request and adds to the context.
func Resolve(c *gin.Context) {
	c.Set("host", resolveHost(c.Request))
	c.Set("scheme", resolveScheme(c.Request))
	c.Next()
}

// parseHeader parses non unique headers value
// from a http.Request and return a slice of the values
// queried from the header
func parseHeader(r *http.Request, header string, token string) (val []string) {
	for _, v := range r.Header[header] {
		options := strings.Split(v, ";")
		for _, o := range options {
			keyvalue := strings.Split(o, "=")
			var key, value string
			if len(keyvalue) > 1 {
				key, value = strings.TrimSpace(keyvalue[0]), strings.TrimSpace(keyvalue[1])
			}
			key = strings.ToLower(key)
			if key == token {
				val = append(val, value)
			}
		}
	}
	return
}

// resolveScheme is a helper function that evaluates the http.Request
// and returns the scheme, HTTP or HTTPS. It is able to detect,
// using the X-Forwarded-Proto, if the original request was HTTPS
// and routed through a reverse proxy with SSL termination.
func resolveScheme(r *http.Request) string {
	switch {
	case r.URL.Scheme == "https":
		return "https"
	case r.TLS != nil:
		return "https"
	case strings.HasPrefix(r.Proto, "HTTPS"):
		return "https"
	case r.Header.Get("X-Forwarded-Proto") == "https":
		return "https"
	case len(r.Header.Get("Forwarded")) != 0 && len(parseHeader(r, "Forwarded", "proto")) != 0 && parseHeader(r, "Forwarded", "proto")[0] == "https":
		return "https"
	default:
		return "http"
	}
}

// resolveHost is a helper function that evaluates the http.Request
// and returns the hostname. It is able to detect, using the
// X-Forarded-For header, the original hostname when routed
// through a reverse proxy.
func resolveHost(r *http.Request) string {
	switch {
	case len(r.Host) != 0:
		return r.Host
	case len(r.URL.Host) != 0:
		return r.URL.Host
	case len(r.Header.Get("X-Forwarded-For")) != 0:
		return r.Header.Get("X-Forwarded-For")
	case len(r.Header.Get("Forwarded")) != 0 && len(parseHeader(r, "Forwarded", "for")) != 0:
		return parseHeader(r, "Forwarded", "for")[0]
	case len(r.Header.Get("X-Host")) != 0:
		return r.Header.Get("X-Host")
	case len(r.Header.Get("Forwarded")) != 0 && len(parseHeader(r, "Forwarded", "host")) != 0:
		return parseHeader(r, "Forwarded", "host")[0]
	case len(r.Header.Get("XFF")) != 0:
		return r.Header.Get("XFF")
	case len(r.Header.Get("X-Real-IP")) != 0:
		return r.Header.Get("X-Real-IP")
	default:
		return "localhost:8000"
	}
}

// Hostname returns the hostname associated with
// the current context.
func Hostname(c *gin.Context) (host string) {
	v, ok := c.Get("host")
	if ok {
		host = v.(string)
	}
	return
}
