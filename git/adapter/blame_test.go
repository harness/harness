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
	"bufio"
	"io"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/types"

	"github.com/google/go-cmp/cmp"
)

func TestBlameReader_NextPart(t *testing.T) {
	// a sample of git blame porcelain output
	const blameOut = `
16f267ad4f731af1b2e36f42e170ed8921377398 9 10 1
author Marko
author-mail <marko.gacesa@harness.io>
author-time 1669812989
author-tz +0100
committer Committer
committer-mail <noreply@harness.io>
committer-time 1669812989
committer-tz +0100
summary Pull request 1
filename file_name_before_rename.go
	Line 10
16f267ad4f731af1b2e36f42e170ed8921377398 12 11 1
	Line 11
dcb4b6b63e86f06ed4e4c52fbc825545dc0b6200 12 12 1
author Marko
author-mail <marko.gacesa@harness.io>
author-time 1673952128
author-tz +0100
committer Committer
committer-mail <noreply@harness.io>
committer-time 1673952128
committer-tz +0100
summary Pull request 2
previous 6561a7b86e1a5e74ea0e4e73ccdfc18b486a2826 file_name.go
filename file_name.go
	Line 12
16f267ad4f731af1b2e36f42e170ed8921377398 13 13 2
	Line 13
16f267ad4f731af1b2e36f42e170ed8921377398 14 14
	Line 14
`

	author := types.Identity{
		Name:  "Marko",
		Email: "marko.gacesa@harness.io",
	}
	committer := types.Identity{
		Name:  "Committer",
		Email: "noreply@harness.io",
	}

	commit1 := &types.Commit{
		SHA:     "16f267ad4f731af1b2e36f42e170ed8921377398",
		Title:   "Pull request 1",
		Message: "",
		Author: types.Signature{
			Identity: author,
			When:     time.Unix(1669812989, 0),
		},
		Committer: types.Signature{
			Identity: committer,
			When:     time.Unix(1669812989, 0),
		},
	}

	commit2 := &types.Commit{
		SHA:     "dcb4b6b63e86f06ed4e4c52fbc825545dc0b6200",
		Title:   "Pull request 2",
		Message: "",
		Author: types.Signature{
			Identity: author,
			When:     time.Unix(1673952128, 0),
		},
		Committer: types.Signature{
			Identity: committer,
			When:     time.Unix(1673952128, 0),
		},
	}

	want := []*types.BlamePart{
		{
			Commit: commit1,
			Lines:  []string{"Line 10", "Line 11"},
		},
		{
			Commit: commit2,
			Lines:  []string{"Line 12"},
		},
		{
			Commit: commit1,
			Lines:  []string{"Line 13", "Line 14"},
		},
	}

	reader := BlameReader{
		scanner:     bufio.NewScanner(strings.NewReader(blameOut)),
		commitCache: make(map[string]*types.Commit),
		errReader:   strings.NewReader(""),
	}

	var got []*types.BlamePart

	for {
		part, err := reader.NextPart()
		if part != nil {
			got = append(got, part)
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				t.Errorf("failed with the error: %v", err)
			}
			break
		}
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(diff)
	}
}

func TestBlameReader_NextPart_UserError(t *testing.T) {
	reader := BlameReader{
		scanner:     bufio.NewScanner(strings.NewReader("")),
		commitCache: make(map[string]*types.Commit),
		errReader:   strings.NewReader("fatal: no such path\n"),
	}

	_, err := reader.NextPart()
	if s := errors.AsStatus(err); s != errors.StatusNotFound {
		t.Errorf("expected NotFound error but got: %v", err)
	}
}

func TestBlameReader_NextPart_CmdError(t *testing.T) {
	reader := BlameReader{
		scanner:     bufio.NewScanner(iotest.ErrReader(errors.New("dummy error"))),
		commitCache: make(map[string]*types.Commit),
		errReader:   strings.NewReader(""),
	}

	_, err := reader.NextPart()
	if s := errors.AsError(err); s.Status != errors.StatusInternal || s.Message != "failed to start git blame command" {
		t.Errorf("expected %v, but got: %v", s.Message, err)
	}
}
