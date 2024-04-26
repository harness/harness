// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"fmt"
	"testing"
	"time"

	"github.com/harness/gitness/git/sha"

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
	require.Equal(t, name, s.Identity.Name, line)
	require.Equal(t, email, s.Identity.Email, line)

	// verify time and offset
	_, offset := s.When.Zone()
	require.Equal(t, expectedTimeUnix, s.When.Unix(), line)
	require.Equal(t, expectedTimeOffset, offset, line)
}

func TestParseTagDataFromCatFile(t *testing.T) {
	when, _ := time.Parse(defaultGitTimeLayout, "Fri Sep 23 10:57:49 2022 -0700")
	testParseTagDataFromCatFileFor(t, sha.EmptyTree.String(), GitObjectTypeTag, "name1",
		Signature{Identity: Identity{Name: "max", Email: "max@mail.com"}, When: when},
		"some message", "some message")

	// test with signature
	testParseTagDataFromCatFileFor(t, sha.EmptyTree.String(), GitObjectTypeCommit, "name2",
		Signature{Identity: Identity{Name: "max", Email: "max@mail.com"}, When: when},
		"gpgsig -----BEGIN PGP SIGNATURE-----\n\nw...B\n-----END PGP SIGNATURE-----\n\nsome message",
		"some message")
}

func testParseTagDataFromCatFileFor(t *testing.T, object string, typ GitObjectType, name string,
	tagger Signature, remainder string, expectedMessage string) {
	data := fmt.Sprintf(
		"object %s\ntype %s\ntag %s\ntagger %s <%s> %s\n%s",
		object, string(typ), name,
		tagger.Identity.Name, tagger.Identity.Email, tagger.When.Format(defaultGitTimeLayout),
		remainder)
	res, err := parseTagDataFromCatFile([]byte(data))
	require.NoError(t, err)

	require.Equal(t, name, res.Name, data)
	require.Equal(t, object, res.TargetSha.String(), data)
	require.Equal(t, typ, res.TargetType, data)
	require.Equal(t, expectedMessage, res.Message, data)
	require.Equal(t, tagger.Identity.Name, res.Tagger.Identity.Name, data)
	require.Equal(t, tagger.Identity.Email, res.Tagger.Identity.Email, data)
	require.Equal(t, tagger.When, res.Tagger.When, data)
}
