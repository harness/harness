package web

import (
	"testing"
)

func Test_injectPartials(t *testing.T) {
	got, want := injectPartials(before), after
	if got != want {
		t.Errorf("Want html %q, got %q", want, got)
	}
}

var before = `<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<!-- drone:version -->
	<!-- drone:user -->
	<!-- drone:csrf -->
<link rel="shortcut icon" href="/favicon.png"></head>
<body>
</html>`

var after = `<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	{{ template "version" . }}
	{{ template "user" . }}
	{{ template "csrf" . }}
<link rel="shortcut icon" href="/favicon.png"></head>
<body>
</html>`
