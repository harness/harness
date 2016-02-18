package client

import (
	"net/url"
	"strings"
)

var encodeMap = map[string]string{
	".": "%252E",
}

func encodeParameter(value string) string {
	value = url.QueryEscape(value)

	for before, after := range encodeMap {
		value = strings.Replace(value, before, after, -1)
	}

	return value
}
