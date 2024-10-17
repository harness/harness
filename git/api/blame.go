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
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/sha"

	"github.com/gotidy/ptr"
)

var (
	// blamePorcelainHeadRE is used to detect line header start in git blame porcelain output.
	// It is explained here: https://www.git-scm.com/docs/git-blame#_the_porcelain_format
	blamePorcelainHeadRE            = regexp.MustCompile(`^([0-9a-f]{40}|[0-9a-f]{64}) (\d+) (\d+)( (\d+))?$`)
	blamePorcelainOutOfRangeErrorRE = regexp.MustCompile(`has only \d+ lines$`)
)

type BlamePart struct {
	Commit   *Commit            `json:"commit"`
	Lines    []string           `json:"lines"`
	Previous *BlamePartPrevious `json:"previous,omitempty"`
}

type BlamePartPrevious struct {
	CommitSHA sha.SHA `json:"commit_sha"`
	FileName  string  `json:"file_name"`
}

type BlameNextReader interface {
	NextPart() (*BlamePart, error)
}

func (g *Git) Blame(
	ctx context.Context,
	repoPath string,
	rev string,
	file string,
	lineFrom int,
	lineTo int,
) BlameNextReader {
	// prepare the git command line arguments
	cmd := command.New(
		"blame",
		command.WithFlag("--porcelain"),
		command.WithFlag("--encoding", "UTF-8"),
	)
	if lineFrom > 0 || lineTo > 0 {
		var lines string
		if lineFrom > 0 {
			lines = strconv.Itoa(lineFrom)
		}
		if lineTo > 0 {
			lines += "," + strconv.Itoa(lineTo)
		}

		cmd.Add(command.WithFlag("-L", lines))
	}

	cmd.Add(command.WithArg(rev))
	cmd.Add(command.WithPostSepArg(file))

	pipeRead, pipeWrite := io.Pipe()
	stderr := &bytes.Buffer{}
	go func() {
		var err error

		defer func() {
			// If running of the command below fails, make the pipe reader also fail with the same error.
			_ = pipeWrite.CloseWithError(err)
		}()

		err = cmd.Run(ctx,
			command.WithDir(repoPath),
			command.WithStdout(pipeWrite),
			command.WithStderr(stderr),
		)
	}()

	return &BlameReader{
		scanner:   bufio.NewScanner(pipeRead),
		cache:     make(map[string]blameReaderCacheItem),
		errReader: stderr, // Any stderr output will cause the BlameReader to fail.
	}
}

type blameReaderCacheItem struct {
	commit   *Commit
	previous *BlamePartPrevious
}

type BlameReader struct {
	scanner   *bufio.Scanner
	lastLine  string
	cache     map[string]blameReaderCacheItem
	errReader io.Reader
}

func (r *BlameReader) nextLine() (string, error) {
	if line := r.lastLine; line != "" {
		r.lastLine = ""
		return line, nil
	}

	for r.scanner.Scan() {
		line := r.scanner.Text()
		if line != "" {
			return line, nil
		}
	}

	if err := r.scanner.Err(); err != nil {
		return "", err
	}

	return "", io.EOF
}

func (r *BlameReader) unreadLine(line string) {
	r.lastLine = line
}

//nolint:complexity,gocognit,nestif // it's ok
func (r *BlameReader) NextPart() (*BlamePart, error) {
	var commit *Commit
	var previous *BlamePartPrevious
	var lines []string
	var err error

	for {
		var line string
		line, err = r.nextLine()
		if err != nil {
			break // This is the only place where we break the loop. Normally it will be the io.EOF.
		}

		if matches := blamePorcelainHeadRE.FindStringSubmatch(line); matches != nil {
			commitSHA := sha.Must(matches[1])

			if commit == nil {
				if cacheItem, ok := r.cache[commitSHA.String()]; ok {
					commit = cacheItem.commit
					previous = cacheItem.previous
				} else {
					commit = &Commit{SHA: commitSHA}
				}

				if matches[5] != "" {
					// At index 5 there's number of lines in this section. However, the resulting
					// BlamePart might contain more than this because we join consecutive sections
					// if the commit SHA is the same.
					lineCount, _ := strconv.Atoi(matches[5])
					lines = make([]string, 0, lineCount)
				}

				continue
			}

			if !commit.SHA.Equal(commitSHA) {
				r.unreadLine(line)
				r.cache[commit.SHA.String()] = blameReaderCacheItem{
					commit:   commit,
					previous: previous,
				}

				return &BlamePart{
					Commit:   commit,
					Lines:    lines,
					Previous: previous,
				}, nil
			}

			continue
		}

		if commit == nil {
			// Continue reading the lines until a line header is reached.
			// This should not happen. Normal output always starts with a line header (with a commit SHA).
			continue
		}

		if line[0] == '\t' {
			// all output that contains actual file data is prefixed with tab, otherwise it's a header line
			lines = append(lines, line[1:])
			continue
		}

		parseBlameHeaders(line, commit, &previous)
	}

	// Check if there's something in the error buffer... If yes, that's the error!
	// It should contain error string from the git.
	errRaw, _ := io.ReadAll(r.errReader)
	if len(errRaw) > 0 {
		line := string(errRaw)

		if idx := bytes.IndexByte(errRaw, '\n'); idx > 0 {
			line = line[:idx] // get only the first line of the output
		}

		line = strings.TrimPrefix(line, "fatal: ") // git errors start with the "fatal: " prefix

		switch {
		case strings.Contains(line, "no such path"):
			return nil, errors.NotFound(line)
		case strings.Contains(line, "bad revision"):
			return nil, errors.NotFound(line)
		case blamePorcelainOutOfRangeErrorRE.MatchString(line):
			return nil, errors.InvalidArgument(line)
		default:
			return nil, errors.Internal(nil, "failed to get next part: %s", line)
		}
	}

	// This error can happen if the command git failed to start. Triggered by pipe writer's CloseWithError call.
	if !errors.Is(err, io.EOF) {
		return nil, errors.Internal(err, "failed to start git blame command")
	}

	var part *BlamePart

	if commit != nil && len(lines) > 0 {
		part = &BlamePart{
			Commit:   commit,
			Lines:    lines,
			Previous: previous,
		}
	}

	return part, err
}

func parseBlameHeaders(line string, commit *Commit, previous **BlamePartPrevious) {
	// This is the list of git blame headers that we process. Other headers we ignore.
	const (
		headerSummary       = "summary "
		headerAuthorName    = "author "
		headerAuthorMail    = "author-mail "
		headerAuthorTime    = "author-time "
		headerCommitterName = "committer "
		headerCommitterMail = "committer-mail "
		headerCommitterTime = "committer-time "
		headerPrevious      = "previous "
	)

	switch {
	case strings.HasPrefix(line, headerSummary):
		commit.Title = extractName(line[len(headerSummary):])
	case strings.HasPrefix(line, headerAuthorName):
		commit.Author.Identity.Name = extractName(line[len(headerAuthorName):])
	case strings.HasPrefix(line, headerAuthorMail):
		commit.Author.Identity.Email = extractEmail(line[len(headerAuthorMail):])
	case strings.HasPrefix(line, headerAuthorTime):
		commit.Author.When = extractTime(line[len(headerAuthorTime):])
	case strings.HasPrefix(line, headerCommitterName):
		commit.Committer.Identity.Name = extractName(line[len(headerCommitterName):])
	case strings.HasPrefix(line, headerCommitterMail):
		commit.Committer.Identity.Email = extractEmail(line[len(headerCommitterMail):])
	case strings.HasPrefix(line, headerCommitterTime):
		commit.Committer.When = extractTime(line[len(headerCommitterTime):])
	case strings.HasPrefix(line, headerPrevious):
		*previous = ptr.Of(extractPrevious(line[len(headerPrevious):]))
	}
}

func extractName(s string) string {
	return s
}

// extractPrevious extracts the sha and filename of the previous commit.
// example: previous 999d2ed306a916423d18e022abe258e92419ab9a README.md
func extractPrevious(s string) BlamePartPrevious {
	rawSHA, fileName, _ := strings.Cut(s, " ")
	if len(fileName) > 0 && fileName[0] == '"' {
		fileName, _ = strconv.Unquote(fileName)
	}
	return BlamePartPrevious{
		CommitSHA: sha.Must(rawSHA),
		FileName:  fileName,
	}
}

// extractEmail extracts email from git blame output.
// The email address is wrapped between "<" and ">" characters.
// If "<" or ">" are not in place it returns the string as it.
func extractEmail(s string) string {
	if len(s) >= 2 && s[0] == '<' && s[len(s)-1] == '>' {
		s = s[1 : len(s)-1]
	}
	return s
}

// extractTime extracts timestamp from git blame output.
// The timestamp is UNIX time (in seconds).
// In case of an error it simply returns zero UNIX time.
func extractTime(s string) time.Time {
	milli, _ := strconv.ParseInt(s, 10, 64)
	return time.Unix(milli, 0)
}
