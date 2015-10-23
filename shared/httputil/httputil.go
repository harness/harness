package httputil

import (
	"net/http"
	"strings"
)

// Struct containing the RFC7239 Forwarded header
type forwardedHeader struct {
	For   []string
	Proto string
	By    []string
	Host  string
}

// parseForwardedHeader parses the RFC7239 Forwarded header from a http.Request
// and return a forwardedHeader to easily access the values
func parseForwardedHeader(r *http.Request) (f forwardedHeader) {
	for _, v := range r.Header["Forwarded"] {
		options := strings.Split(v, ";")
		for _, o := range options {
			keyvalue := strings.Split(o, "=")
			key, value := strings.TrimSpace(keyvalue[0]), strings.TrimSpace(keyvalue[1])

			key = strings.ToLower(key)
			switch key {
			case "for":
				f.For = append(f.For, value)
			case "proto":
				f.Proto = value
			case "by":
				f.By = append(f.By, value)
			case "host":
				f.Host = value
			}
		}
	}
	return
}

func hasHttpsForwarded(r *http.Request) bool {
	forwardedHeader := parseForwardedHeader(r)
	if forwardedHeader.Proto == "https" {
		return true
	}
	return false
}

// IsHttps is a helper function that evaluates the http.Request
// and returns True if the Request uses HTTPS. It is able to detect,
// using the X-Forwarded-Proto, if the original request was HTTPS and
// routed through a reverse proxy with SSL termination.
func IsHttps(r *http.Request) bool {
	switch {
	case r.URL.Scheme == "https":
		return true
	case r.TLS != nil:
		return true
	case strings.HasPrefix(r.Proto, "HTTPS"):
		return true
	case r.Header.Get("X-Forwarded-Proto") == "https":
		return true
	default:
		return false
	}
}

// GetScheme is a helper function that evaluates the http.Request
// and returns the scheme, HTTP or HTTPS. It is able to detect,
// using the X-Forwarded-Proto, if the original request was HTTPS
// and routed through a reverse proxy with SSL termination.
func GetScheme(r *http.Request) string {
	switch {
	case r.URL.Scheme == "https":
		return "https"
	case r.TLS != nil:
		return "https"
	case strings.HasPrefix(r.Proto, "HTTPS"):
		return "https"
	case r.Header.Get("X-Forwarded-Proto") == "https":
		return "https"
	default:
		return "http"
	}
}

// GetHost is a helper function that evaluates the http.Request
// and returns the hostname. It is able to detect, using the
// X-Forarded-For header, the original hostname when routed
// through a reverse proxy.
func GetHost(r *http.Request) string {
	forwardedHeader := parseForwardedHeader(r)
	switch {
	case len(r.Host) != 0:
		return r.Host
	case len(r.URL.Host) != 0:
		return r.URL.Host
	case len(forwardedHeader.For) != 0:
		return forwardedHeader.For[0]
	case forwardedHeader.Host != "":
		return forwardedHeader.Host
	case len(r.Header.Get("XFF")) != 0:
		return r.Header.Get("XFF")
	default:
		return "localhost:8080"
	}
}

// GetURL is a helper function that evaluates the http.Request
// and returns the URL as a string. Only the scheme + hostname
// are included; the path is excluded.
func GetURL(r *http.Request) string {
	return GetScheme(r) + "://" + GetHost(r)
}

// GetCookie retrieves and verifies the cookie value.
func GetCookie(r *http.Request, name string) (value string) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return
	}
	value = cookie.Value
	return
}

// SetCookie writes the cookie value.
func SetCookie(w http.ResponseWriter, r *http.Request, name, value string) {
	cookie := http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   r.URL.Host,
		HttpOnly: true,
		Secure:   IsHttps(r),
		MaxAge:   2147483647, // the cooke value (token) is responsible for expiration
	}

	http.SetCookie(w, &cookie)
}

// DelCookie deletes a cookie.
func DelCookie(w http.ResponseWriter, r *http.Request, name string) {
	cookie := http.Cookie{
		Name:   name,
		Value:  "deleted",
		Path:   "/",
		Domain: r.URL.Host,
		MaxAge: -1,
	}

	http.SetCookie(w, &cookie)
}
