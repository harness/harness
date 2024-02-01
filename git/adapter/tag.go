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

package adapter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/types"
)

const (
	pgpSignatureBeginToken = "\n-----BEGIN PGP SIGNATURE-----\n" //#nosec G101
	pgpSignatureEndToken   = "\n-----END PGP SIGNATURE-----"     //#nosec G101
)

// GetAnnotatedTag returns the tag for a specific tag sha.
func (a Adapter) GetAnnotatedTag(
	ctx context.Context,
	repoPath string,
	sha string,
) (*types.Tag, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}
	tags, err := getAnnotatedTags(ctx, repoPath, []string{sha})
	if err != nil || len(tags) == 0 {
		return nil, processGiteaErrorf(err, "failed to get annotated tag with sha '%s'", sha)
	}

	return &tags[0], nil
}

// GetAnnotatedTags returns the tags for a specific list of tag sha.
func (a Adapter) GetAnnotatedTags(
	ctx context.Context,
	repoPath string,
	shas []string,
) ([]types.Tag, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}
	return getAnnotatedTags(ctx, repoPath, shas)
}

// CreateTag creates the tag pointing at the provided SHA (could be any type, e.g. commit, tag, blob, ...)
func (a Adapter) CreateTag(
	ctx context.Context,
	repoPath string,
	name string,
	targetSHA string,
	opts *types.CreateTagOptions,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}
	cmd := command.New("tag")
	if opts != nil && opts.Message != "" {
		cmd.Add(command.WithFlag("-m", opts.Message))
		cmd.Add(
			command.WithCommitterAndDate(
				opts.Tagger.Identity.Name,
				opts.Tagger.Identity.Email,
				opts.Tagger.When,
			),
		)
	}

	cmd.Add(command.WithArg(name, targetSHA))
	err := cmd.Run(ctx, command.WithDir(repoPath))
	if err != nil {
		return processGiteaErrorf(err, "Service failed to create a tag")
	}
	return nil
}

// getAnnotatedTag is a custom implementation to retrieve an annotated tag from a sha.
// The code is following parts of the gitea implementation.
func getAnnotatedTags(
	ctx context.Context,
	repoPath string,
	shas []string,
) ([]types.Tag, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}
	// The tag is an annotated tag with a message.
	writer, reader, cancel := CatFileBatch(ctx, repoPath)
	defer func() {
		cancel()
		_ = writer.Close()
	}()

	tags := make([]types.Tag, len(shas))

	for i, sha := range shas {
		if _, err := writer.Write([]byte(sha + "\n")); err != nil {
			return nil, err
		}
		tagSha, typ, size, err := ReadBatchHeaderLine(reader)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.IsNotFound(err) {
				return nil, fmt.Errorf("tag with sha %s does not exist", sha)
			}
			return nil, err
		}
		if typ != string(types.GitObjectTypeTag) {
			return nil, fmt.Errorf("git object is of type '%s', expected tag", typ)
		}

		// read the remaining rawData
		rawData, err := io.ReadAll(io.LimitReader(reader, size))
		if err != nil {
			return nil, err
		}
		_, err = reader.Discard(1)
		if err != nil {
			return nil, err
		}

		tag, err := parseTagDataFromCatFile(rawData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse tag '%s': %w", sha, err)
		}

		// fill in the sha
		tag.Sha = string(tagSha)

		tags[i] = tag
	}

	return tags, nil
}

// parseTagDataFromCatFile parses a tag from a cat-file output.
func parseTagDataFromCatFile(data []byte) (tag types.Tag, err error) {
	p := 0

	// parse object Id
	tag.TargetSha, p, err = parseCatFileLine(data, p, "object")
	if err != nil {
		return tag, err
	}

	// parse object type
	rawType, p, err := parseCatFileLine(data, p, "type")
	if err != nil {
		return tag, err
	}

	tag.TargetType, err = types.ParseGitObjectType(rawType)
	if err != nil {
		return tag, err
	}

	// parse tag name
	tag.Name, p, err = parseCatFileLine(data, p, "tag")
	if err != nil {
		return tag, err
	}

	// parse tagger
	rawTaggerInfo, p, err := parseCatFileLine(data, p, "tagger")
	if err != nil {
		return tag, err
	}
	tag.Tagger, err = parseSignatureFromCatFileLine(rawTaggerInfo)
	if err != nil {
		return tag, err
	}

	// remainder is message and gpg (remove leading and tailing new lines)
	message := string(bytes.Trim(data[p:], "\n"))

	// handle gpg signature
	pgpEnd := strings.Index(message, pgpSignatureEndToken)
	if pgpEnd > -1 {
		messageStart := pgpEnd + len(pgpSignatureEndToken)
		// for now we just remove the signature (and trim any separating new lines)
		// TODO: add support for GPG signature of tags
		message = strings.TrimLeft(message[messageStart:], "\n")
	}

	tag.Message = message

	// get title from message
	tag.Title = message
	titleEnd := strings.IndexByte(message, '\n')
	if titleEnd > -1 {
		tag.Title = message[:titleEnd]
	}

	return tag, nil
}

func parseCatFileLine(data []byte, start int, header string) (string, int, error) {
	// for simplicity only look at data from start onwards
	data = data[start:]

	lenHeader := len(header)
	lenData := len(data)
	if lenData < lenHeader {
		return "", 0, fmt.Errorf("expected '%s' but line only contains '%s'", header, string(data))
	}
	if string(data[:lenHeader]) != header {
		return "", 0, fmt.Errorf("expected '%s' but started with '%s'", header, string(data[:lenHeader]))
	}

	// get end of line and start of next line (used externally, transpose with provided start index)
	lineEnd := bytes.IndexByte(data, '\n')
	externalNextLine := start + lineEnd + 1
	if lineEnd == -1 {
		lineEnd = lenData
		externalNextLine = start + lenData
	}

	// if there's no data, return an error (have to consider for ' ')
	if lineEnd <= lenHeader+1 {
		return "", 0, fmt.Errorf("no data for line of type '%s'", header)
	}

	return string(data[lenHeader+1 : lineEnd]), externalNextLine, nil
}

// defaultGitTimeLayout is the (default) time format printed by git.
const defaultGitTimeLayout = "Mon Jan _2 15:04:05 2006 -0700"

// parseSignatureFromCatFileLine parses the signature from a cat-file output.
// This is used for commit / tag outputs. Input will be similar to (without 'author 'prefix):
// - author Max Mustermann <mm@gitness.io> 1666401234 -0700
// - author Max Mustermann <mm@gitness.io> Tue Oct 18 05:13:26 2022 +0530
// TODO: method is leaning on gitea code - requires reference?
func parseSignatureFromCatFileLine(line string) (types.Signature, error) {
	sig := types.Signature{}
	emailStart := strings.LastIndexByte(line, '<')
	emailEnd := strings.LastIndexByte(line, '>')
	if emailStart == -1 || emailEnd == -1 || emailEnd < emailStart {
		return types.Signature{}, fmt.Errorf("signature is missing email ('%s')", line)
	}

	// name requires that there is at least one char followed by a space (so emailStart >= 2)
	if emailStart < 2 {
		return types.Signature{}, fmt.Errorf("signature is missing name ('%s')", line)
	}

	sig.Identity.Name = line[:emailStart-1]
	sig.Identity.Email = line[emailStart+1 : emailEnd]

	timeStart := emailEnd + 2
	if timeStart >= len(line) {
		return types.Signature{}, fmt.Errorf("signature is missing time ('%s')", line)
	}

	// Check if time format is written date time format (e.g Thu, 07 Apr 2005 22:13:13 +0200)
	// we can check that by ensuring that the date time part starts with a non-digit character.
	if line[timeStart] > '9' {
		var err error
		sig.When, err = time.Parse(defaultGitTimeLayout, line[timeStart:])
		if err != nil {
			return types.Signature{}, fmt.Errorf("failed to time.parse signature time ('%s'): %w", line, err)
		}

		return sig, nil
	}

	// Otherwise we have to manually parse unix time and time zone
	endOfUnixTime := timeStart + strings.IndexByte(line[timeStart:], ' ')
	if endOfUnixTime <= timeStart {
		return types.Signature{}, fmt.Errorf("signature is missing unix time ('%s')", line)
	}

	unixSeconds, err := strconv.ParseInt(line[timeStart:endOfUnixTime], 10, 64)
	if err != nil {
		return types.Signature{}, fmt.Errorf("failed to parse unix time ('%s'): %w", line, err)
	}

	// parse time zone
	startOfTimeZone := endOfUnixTime + 1 // +1 for space
	endOfTimeZone := startOfTimeZone + 5 // +5 for '+0700'
	if startOfTimeZone >= len(line) || endOfTimeZone > len(line) {
		return types.Signature{}, fmt.Errorf("signature is missing time zone ('%s')", line)
	}

	// get and disect timezone, e.g. '+0700'
	rawTimeZone := line[startOfTimeZone:endOfTimeZone]
	rawTimeZoneH := rawTimeZone[1:3]  // gets +[07]00
	rawTimeZoneMin := rawTimeZone[3:] // gets +07[00]
	timeZoneH, err := strconv.ParseInt(rawTimeZoneH, 10, 64)
	if err != nil {
		return types.Signature{}, fmt.Errorf("failed to parse hours of time zone ('%s'): %w", line, err)
	}
	timeZoneMin, err := strconv.ParseInt(rawTimeZoneMin, 10, 64)
	if err != nil {
		return types.Signature{}, fmt.Errorf("failed to parse minutes of time zone ('%s'): %w", line, err)
	}

	timeZoneOffsetInSec := int(timeZoneH*60+timeZoneMin) * 60
	if rawTimeZone[0] == '-' {
		timeZoneOffsetInSec *= -1
	}
	timeZone := time.FixedZone("", timeZoneOffsetInSec)

	// create final time using unix and timezone translation
	sig.When = time.Unix(unixSeconds, 0).In(timeZone)

	return sig, nil
}
