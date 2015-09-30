// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssh

import (
	"bytes"
	"strings"
	"testing"
)

func TestReadVersion(t *testing.T) {
	longversion := strings.Repeat("SSH-2.0-bla", 50)[:253]
	cases := map[string]string{
		"SSH-2.0-bla\r\n":    "SSH-2.0-bla",
		"SSH-2.0-bla\n":      "SSH-2.0-bla",
		longversion + "\r\n": longversion,
	}

	for in, want := range cases {
		result, err := readVersion(bytes.NewBufferString(in))
		if err != nil {
			t.Errorf("readVersion(%q): %s", in, err)
		}
		got := string(result)
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

func TestReadVersionError(t *testing.T) {
	longversion := strings.Repeat("SSH-2.0-bla", 50)[:253]
	cases := []string{
		longversion + "too-long\r\n",
	}
	for _, in := range cases {
		if _, err := readVersion(bytes.NewBufferString(in)); err == nil {
			t.Errorf("readVersion(%q) should have failed", in)
		}
	}
}

func TestExchangeVersionsBasic(t *testing.T) {
	v := "SSH-2.0-bla"
	buf := bytes.NewBufferString(v + "\r\n")
	them, err := exchangeVersions(buf, []byte("xyz"))
	if err != nil {
		t.Errorf("exchangeVersions: %v", err)
	}

	if want := "SSH-2.0-bla"; string(them) != want {
		t.Errorf("got %q want %q for our version", them, want)
	}
}

func TestExchangeVersions(t *testing.T) {
	cases := []string{
		"not\x000allowed",
		"not allowed\n",
	}
	for _, c := range cases {
		buf := bytes.NewBufferString("SSH-2.0-bla\r\n")
		if _, err := exchangeVersions(buf, []byte(c)); err == nil {
			t.Errorf("exchangeVersions(%q): should have failed", c)
		}
	}
}
