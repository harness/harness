// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseSignatureFromCatFileLine(t *testing.T) {
	// test good cases
	testParseSignatureFromCatFileLineFor(t, "Max Mustermann", "max.mm@me.io", "1666401234 -0700", 1666401234, -7*60*60)
	testParseSignatureFromCatFileLineFor(t, "Max", "max@gitness.io", "1666050206 +0530", 1666050206, 5*60*60+30*60)
	testParseSignatureFromCatFileLineFor(t, "Max", "max@gitness.io", "1666401234 +0000", 1666401234, 0)
	testParseSignatureFromCatFileLineFor(t, "Max", "randomEmail", "1666401234 -0000", 1666401234, 0)
	testParseSignatureFromCatFileLineFor(t, "Max", "max@mm.io", "Fri Sep 23 10:57:49 2022 -0700", 1663955869, -7*60*60)

	// test bad cases
	_, err := parseSignatureFromCatFileLine("<email> 1666401234 -0700")
	require.Error(t, err, "no name")
	_, err = parseSignatureFromCatFileLine("name 1666401234 -0700")
	require.Error(t, err, "no email")
	_, err = parseSignatureFromCatFileLine("name <email>")
	require.Error(t, err, "no time")
	_, err = parseSignatureFromCatFileLine("name <email> ")
	require.Error(t, err, "no time2")
	_, err = parseSignatureFromCatFileLine("name <email> 1666050206")
	require.Error(t, err, "no timezone with unix")
	_, err = parseSignatureFromCatFileLine("name <email> +0800")
	require.Error(t, err, "no unix with timezone")
	_, err = parseSignatureFromCatFileLine("name <email> 1666050206 0800")
	require.Error(t, err, "timezone no sign")
	_, err = parseSignatureFromCatFileLine("name <email> 1666050206 +080")
	require.Error(t, err, "timezone too short")
	_, err = parseSignatureFromCatFileLine("name <email> 1666050206 +00a0")
	require.Error(t, err, "timezone invald char")
}

func testParseSignatureFromCatFileLineFor(t *testing.T, name string, email string, timeAsString string,
	expectedTimeUnix int64, expectedTimeOffset int) {
	line := fmt.Sprintf("%s <%s> %s", name, email, timeAsString)
	s, err := parseSignatureFromCatFileLine(line)

	require.NoError(t, err, line)
	require.Equal(t, name, s.identity.name, line)
	require.Equal(t, email, s.identity.email, line)

	// verify time and offset
	_, offset := s.when.Zone()
	require.Equal(t, expectedTimeUnix, s.when.Unix(), line)
	require.Equal(t, expectedTimeOffset, offset, line)
}

func TestParseTagDataFromCatFile(t *testing.T) {
	when, _ := time.Parse(defaultGitTimeLayout, "Fri Sep 23 10:57:49 2022 -0700")
	testParseTagDataFromCatFileFor(t, "sha012", gitObjectTypeTag, "name1",
		signature{identity: identity{name: "max", email: "max@mail.com"}, when: when},
		"some message", "some message")

	// test with signature
	testParseTagDataFromCatFileFor(t, "sha012", gitObjectTypeCommit, "name2",
		signature{identity: identity{name: "max", email: "max@mail.com"}, when: when},
		"gpgsig -----BEGIN PGP SIGNATURE-----\n\nw...B\n-----END PGP SIGNATURE-----\n\nsome message", "some message")
}

func testParseTagDataFromCatFileFor(t *testing.T, object string, typ gitObjectType, name string, tagger signature,
	remainder string, expectedMessage string) {
	data := fmt.Sprintf(
		"object %s\ntype %s\ntag %s\ntagger %s <%s> %s\n%s",
		object, string(typ), name,
		tagger.identity.name, tagger.identity.email, tagger.when.Format(defaultGitTimeLayout),
		remainder)
	res, err := parseTagDataFromCatFile([]byte(data))
	require.NoError(t, err)

	require.Equal(t, name, res.name, data)
	require.Equal(t, object, res.targetSha, data)
	require.Equal(t, typ, res.targetType, data)
	require.Equal(t, expectedMessage, res.message, data)
	require.Equal(t, tagger.identity.name, res.tagger.identity.name, data)
	require.Equal(t, tagger.identity.email, res.tagger.identity.email, data)
	require.Equal(t, tagger.when, res.tagger.when, data)
}
