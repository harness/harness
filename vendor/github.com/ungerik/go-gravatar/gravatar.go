/*
Wrapper for the Gravatar API.

See: http://en.gravatar.com/site/implement/
*/
package gravatar

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Default images (used as defaultURL)
const (
	HTTP404    = "404"       // do not load any image if none is associated with the email hash, instead return an HTTP 404 (File Not Found) response
	MysteryMan = "mm"        // (mystery-man) a simple, cartoon-style silhouetted outline of a person (does not vary by email hash)
	IdentIcon  = "identicon" // a geometric pattern based on an email hash
	MonsterID  = "monsterid" // a generated 'monster' with different colors, faces, etc
	Wavatar    = "wavatar"   // generated faces with differing features and backgrounds
	Retro      = "retro"     // awesome generated, 8-bit arcade-style pixelated faces
)

func Hash(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	hash := md5.New()
	hash.Write([]byte(email))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func Url(email string) string {
	return "http://www.gravatar.com/avatar/" + Hash(email)
}

func UrlDefault(email, defaultURL string) string {
	return Url(email) + "?d=" + url.QueryEscape(defaultURL)
}

// You may request images anywhere from 1px up to 512px
func UrlSize(email string, size int) string {
	return fmt.Sprintf("http://www.gravatar.com/avatar/%s?s=%d", Hash(email), size)
}

func UrlSizeDefault(email string, size int, defaultURL string) string {
	return UrlSize(email, size) + "&d=" + url.QueryEscape(defaultURL)
}

func SecureUrl(email string) string {
	return "https://secure.gravatar.com/avatar/" + Hash(email)
}

func SecureUrlDefault(email, defaultURL string) string {
	return SecureUrl(email) + "?d=" + url.QueryEscape(defaultURL)
}

func SecureUrlSize(email string, size int) string {
	return fmt.Sprintf("https://secure.gravatar.com/avatar/%s?s=%d", Hash(email), size)
}

func SecureUrlSizeDefault(email string, size int, defaultURL string) string {
	return SecureUrlSize(email, size) + "&d=" + url.QueryEscape(defaultURL)
}

func Available(email string) (ok bool, err error) {
	url := fmt.Sprintf("http://www.gravatar.com/avatar/%s?d=404", Hash(email))
	response, err := http.Get(url)
	if err != nil {
		return false, err
	}
	return response.StatusCode != 404, nil
}
