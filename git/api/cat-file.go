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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"iter"
	"strconv"
	"strings"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/parser"
	"github.com/harness/gitness/git/sha"

	"github.com/djherbis/buffer"
	"github.com/djherbis/nio/v3"
)

// WriteCloserError wraps an io.WriteCloser with an additional CloseWithError function.
type WriteCloserError interface {
	io.WriteCloser
	CloseWithError(err error) error
}

// CatFileBatch opens git cat-file --batch in the provided repo and returns a stdin pipe,
// a stdout reader and cancel function.
func CatFileBatch(
	ctx context.Context,
	repoPath string,
	alternateObjectDirs []string,
	flags ...command.CmdOptionFunc,
) (WriteCloserError, *bufio.Reader, func()) {
	const bufferSize = 32 * 1024
	// We often want to feed the commits in order into cat-file --batch,
	// followed by their trees and sub trees as necessary.
	batchStdinReader, batchStdinWriter := io.Pipe()
	batchStdoutReader, batchStdoutWriter := nio.Pipe(buffer.New(bufferSize))
	ctx, ctxCancel := context.WithCancel(ctx)
	closed := make(chan struct{})
	cancel := func() {
		ctxCancel()
		_ = batchStdinWriter.Close()
		_ = batchStdoutReader.Close()
		<-closed
	}

	// Ensure cancel is called as soon as the provided context is cancelled
	go func() {
		<-ctx.Done()
		cancel()
	}()

	go func() {
		stderr := bytes.Buffer{}
		cmd := command.New("cat-file",
			command.WithFlag("--batch"),
			command.WithAlternateObjectDirs(alternateObjectDirs...),
		)
		cmd.Add(flags...)
		err := cmd.Run(ctx,
			command.WithDir(repoPath),
			command.WithStdin(batchStdinReader),
			command.WithStdout(batchStdoutWriter),
			command.WithStderr(&stderr),
		)
		if err != nil {
			_ = batchStdoutWriter.CloseWithError(command.NewError(err, stderr.Bytes()))
			_ = batchStdinReader.CloseWithError(command.NewError(err, stderr.Bytes()))
		} else {
			_ = batchStdoutWriter.Close()
			_ = batchStdinReader.Close()
		}
		close(closed)
	}()

	// For simplicities sake we'll us a buffered reader to read from the cat-file --batch
	batchReader := bufio.NewReaderSize(batchStdoutReader, bufferSize)

	return batchStdinWriter, batchReader, cancel
}

type BatchHeaderResponse struct {
	SHA  sha.SHA
	Type GitObjectType
	Size int64
}

// ReadBatchHeaderLine reads the header line from cat-file --batch
// <sha> SP <type> SP <size> LF
// sha is a 40byte not 20byte here.
func ReadBatchHeaderLine(rd *bufio.Reader) (*BatchHeaderResponse, error) {
	line, err := rd.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if len(line) == 1 {
		line, err = rd.ReadString('\n')
		if err != nil {
			return nil, err
		}
	}
	idx := strings.IndexByte(line, ' ')
	if idx < 0 {
		return nil, errors.NotFound("missing space char for: %s", line)
	}
	id := line[:idx]
	objType := line[idx+1:]

	if objType == "missing" {
		return nil, errors.NotFound("sha '%s' not found", id)
	}

	idx = strings.IndexByte(objType, ' ')
	if idx < 0 {
		return nil, errors.NotFound("sha '%s' not found", id)
	}

	sizeStr := objType[idx+1 : len(objType)-1]
	objType = objType[:idx]

	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return nil, err
	}
	return &BatchHeaderResponse{
		SHA:  sha.Must(id),
		Type: GitObjectType(objType),
		Size: size,
	}, nil
}

func catFileObjects(
	ctx context.Context,
	repoPath string,
	alternateObjectDirs []string,
	iter iter.Seq[string],
	fn func(string, sha.SHA, GitObjectType, []byte) error,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}

	writer, reader, cancel := CatFileBatch(ctx, repoPath, alternateObjectDirs)
	defer func() {
		cancel()
		_ = writer.Close()
	}()

	buf := bytes.NewBuffer(nil)
	buf.Grow(512)

	for objectName := range iter {
		if _, err := writer.Write([]byte(objectName)); err != nil {
			return fmt.Errorf("failed to write object sha to cat-file stdin: %w", err)
		}

		if _, err := writer.Write([]byte{'\n'}); err != nil {
			return fmt.Errorf("failed to write EOL to cat-file stdin: %w", err)
		}

		output, err := ReadBatchHeaderLine(reader)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.IsNotFound(err) {
				return fmt.Errorf("object %s does not exist", objectName)
			}
			return fmt.Errorf("failed to read cat-file object header line: %w", err)
		}

		buf.Reset()

		_, err = io.CopyN(buf, reader, output.Size)
		if err != nil {
			return fmt.Errorf("failed to read object raw data: %w", err)
		}

		_, err = reader.Discard(1)
		if err != nil {
			return fmt.Errorf("failed to discard EOL byte: %w", err)
		}

		err = fn(objectName, output.SHA, output.Type, buf.Bytes())
		if err != nil {
			return fmt.Errorf("failed to parse %s with sha %s: %w", output.Type, output.SHA.String(), err)
		}
	}

	return nil
}

func CatFileCommits(
	ctx context.Context,
	repoPath string,
	alternateObjectDirs []string,
	commitSHAs []sha.SHA,
) ([]Commit, error) {
	commits := make([]Commit, 0, len(commitSHAs))

	seq := func(yield func(string) bool) {
		for _, commitSHA := range commitSHAs {
			if !yield(commitSHA.String()) {
				return
			}
		}
	}

	err := catFileObjects(ctx, repoPath, alternateObjectDirs, seq, func(
		_ string,
		commitSHA sha.SHA,
		objectType GitObjectType,
		raw []byte,
	) error {
		if objectType != GitObjectTypeCommit {
			return fmt.Errorf("for SHA %s expected commit, but received %q", commitSHA.String(), objectType)
		}

		rawObject, err := parser.Object(raw)
		if err != nil {
			return fmt.Errorf("failed to parse git object for SHA %s: %w", commitSHA.String(), err)
		}

		commit, err := asCommit(commitSHA, rawObject)
		if err != nil {
			return fmt.Errorf("failed to convert git object to commit SHA %s: %w", commitSHA.String(), err)
		}

		commits = append(commits, commit)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to cat-file commits: %w", err)
	}

	return commits, nil
}

func CatFileAnnotatedTag(
	ctx context.Context,
	repoPath string,
	alternateObjectDirs []string,
	rev string,
) (*Tag, error) {
	var tags []Tag

	seq := func(yield func(string) bool) { yield(rev) }

	tags, err := catFileAnnotatedTags(ctx, repoPath, alternateObjectDirs, tags, seq)
	if err != nil {
		return nil, err
	}

	return &tags[0], nil
}

func CatFileAnnotatedTagFromSHAs(
	ctx context.Context,
	repoPath string,
	alternateObjectDirs []string,
	tagSHAs []sha.SHA,
) ([]Tag, error) {
	tags := make([]Tag, 0, len(tagSHAs))

	seq := func(yield func(string) bool) {
		for _, tagSHA := range tagSHAs {
			if !yield(tagSHA.String()) {
				return
			}
		}
	}

	return catFileAnnotatedTags(ctx, repoPath, alternateObjectDirs, tags, seq)
}

func catFileAnnotatedTags(
	ctx context.Context,
	repoPath string,
	alternateObjectDirs []string,
	tags []Tag,
	seq iter.Seq[string],
) ([]Tag, error) {
	err := catFileObjects(ctx, repoPath, alternateObjectDirs, seq, func(
		objectName string,
		tagSHA sha.SHA,
		objectType GitObjectType,
		raw []byte,
	) error {
		if objectType != GitObjectTypeTag {
			return fmt.Errorf("for %q (%s) expected tag, but received %q",
				objectName, tagSHA.String(), objectType)
		}

		rawObject, err := parser.Object(raw)
		if err != nil {
			return fmt.Errorf("failed to parse git object %s: %w", tagSHA.String(), err)
		}

		tag, err := asTag(tagSHA, rawObject)
		if err != nil {
			return fmt.Errorf("failed to convert git object %s to tag: %w", tagSHA.String(), err)
		}

		tags = append(tags, tag)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to cat-file commits: %w", err)
	}

	return tags, nil
}

func asCommit(
	commitSHA sha.SHA,
	raw parser.ObjectRaw,
) (Commit, error) {
	var (
		treeCount      int
		authorCount    int
		committerCount int

		treeSHA    sha.SHA
		parentSHAs []sha.SHA
		author     Signature
		committer  Signature
	)

	for _, header := range raw.Headers {
		value := strings.TrimRight(header.Value, "\n")

		switch header.Type {
		case "tree":
			entrySHA, err := sha.New(strings.TrimSpace(value))
			if err != nil {
				return Commit{}, fmt.Errorf("failed to parse tree SHA %s: %w", value, err)
			}
			treeCount++
			treeSHA = entrySHA
		case "parent":
			entrySHA, err := sha.New(value)
			if err != nil {
				return Commit{}, fmt.Errorf("failed to parse parent SHA %s: %w", value, err)
			}
			parentSHAs = append(parentSHAs, entrySHA)
		case "author":
			name, email, when, err := parser.ObjectHeaderIdentity(value)
			if err != nil {
				return Commit{}, fmt.Errorf("failed to parse author SHA %s: %w", value, err)
			}
			authorCount++
			author = Signature{Identity: Identity{Name: name, Email: email}, When: when}
		case "committer":
			name, email, when, err := parser.ObjectHeaderIdentity(value)
			if err != nil {
				return Commit{}, fmt.Errorf("failed to parse committer SHA %s: %w", value, err)
			}
			committerCount++
			committer = Signature{Identity: Identity{Name: name, Email: email}, When: when}
		case "encoding":
			// encoding is not currently processed - we assume UTF-8
		case "mergetag":
			// encoding is not currently processed
		default:
			// custom headers not processed.
		}
	}

	if treeCount != 1 && authorCount != 1 && committerCount != 1 {
		return Commit{}, fmt.Errorf("incomplete commit info: trees=%d authors=%d committers=%d",
			treeCount, authorCount, committerCount)
	}

	title := parser.ExtractSubject(raw.Message)

	var signedData *SignedData
	if raw.SignatureType != "" {
		signedData = &SignedData{
			Type:          raw.SignatureType,
			Signature:     raw.Signature,
			SignedContent: raw.SignedContent,
		}
	}

	return Commit{
		SHA:        commitSHA,
		TreeSHA:    treeSHA,
		ParentSHAs: parentSHAs,
		Title:      title,
		Message:    raw.Message,
		Author:     author,
		Committer:  committer,
		SignedData: signedData,
		FileStats:  nil,
	}, nil
}

func asTag(tagSHA sha.SHA, raw parser.ObjectRaw) (Tag, error) {
	var (
		objectCount  int
		tagTypeCount int
		tagNameCount int
		taggerCount  int

		objectSHA sha.SHA
		tagType   GitObjectType
		tagName   string
		tagger    Signature
	)

	for _, header := range raw.Headers {
		value := strings.TrimRight(header.Value, "\n")
		switch header.Type {
		case "object":
			entrySHA, err := sha.New(value)
			if err != nil {
				return Tag{}, fmt.Errorf("failed to parse object SHA %s: %w", value, err)
			}
			objectCount++
			objectSHA = entrySHA
		case "type":
			entryType, err := ParseGitObjectType(value)
			if err != nil {
				return Tag{}, err
			}
			tagTypeCount++
			tagType = entryType
		case "tag":
			tagNameCount++
			tagName = value
		case "tagger":
			name, email, when, err := parser.ObjectHeaderIdentity(value)
			if err != nil {
				return Tag{}, fmt.Errorf("failed to parse tagger %s: %w", value, err)
			}
			taggerCount++
			tagger = Signature{Identity: Identity{Name: name, Email: email}, When: when}
		default:
		}
	}

	if objectCount != 1 && tagTypeCount != 1 && tagNameCount != 1 {
		return Tag{}, fmt.Errorf("incomplete tag info: objects=%d types=%d tags=%d",
			objectCount, tagTypeCount, tagNameCount)
	}

	title := parser.ExtractSubject(raw.Message)

	var signedData *SignedData
	if raw.SignatureType != "" {
		signedData = &SignedData{
			Type:          raw.SignatureType,
			Signature:     raw.Signature,
			SignedContent: raw.SignedContent,
		}
	}

	return Tag{
		Sha:        tagSHA,
		Name:       tagName,
		TargetSHA:  objectSHA,
		TargetType: tagType,
		Title:      title,
		Message:    raw.Message,
		Tagger:     tagger,
		SignedData: signedData,
	}, nil
}
