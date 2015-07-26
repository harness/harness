package gogitlab

import (
	"net/url"
	"strings"
)

func encodeParameter(value string) string {
	return strings.Replace(url.QueryEscape(value), "/", "%2F", 0)
}
