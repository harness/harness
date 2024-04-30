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

package sharedrepo

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/harness/gitness/git/parser"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parsePatchTextFilePayloads(t *testing.T) {
	tests := []struct {
		name    string
		arg     [][]byte
		wantErr string
		want    []patchTextFileReplacement
	}{
		{
			name: "test no payloads",
			arg:  nil,
			want: []patchTextFileReplacement{},
		},
		{
			name: "test no zero byte splitter",
			arg: [][]byte{
				[]byte("0:1"),
			},
			wantErr: "Payload format is missing the content separator",
		},
		{
			name: "test line range wrong format",
			arg: [][]byte{
				[]byte("0\u0000"),
			},
			wantErr: "Payload is missing the line number separator",
		},
		{
			name: "test start line error returned",
			arg: [][]byte{
				[]byte("0:1\u0000"),
			},
			wantErr: "Payload start line number is invalid",
		},
		{
			name: "test end line error returned",
			arg: [][]byte{
				[]byte("1:a\u0000"),
			},
			wantErr: "Payload end line number is invalid",
		},
		{
			name: "test end smaller than start",
			arg: [][]byte{
				[]byte("2:1\u0000"),
			},
			wantErr: "Payload end line has to be at least as big as start line",
		},
		{
			name: "payload empty",
			arg: [][]byte{
				[]byte("1:2\u0000"),
			},
			want: []patchTextFileReplacement{
				{
					OmitFrom:     1,
					ContinueFrom: 2,
					Content:      []byte{},
				},
			},
		},
		{
			name: "payload non-empty with zero byte and line endings",
			arg: [][]byte{
				[]byte("1:eof\u0000a\nb\r\nc\u0000d"),
			},
			want: []patchTextFileReplacement{
				{
					OmitFrom:     1,
					ContinueFrom: lineNumberEOF,
					Content:      []byte("a\nb\r\nc\u0000d"),
				},
			},
		},
		{
			name: "multiple payloads",
			arg: [][]byte{
				[]byte("1:3\u0000a"),
				[]byte("2:eof\u0000b"),
			},
			want: []patchTextFileReplacement{
				{
					OmitFrom:     1,
					ContinueFrom: 3,
					Content:      []byte("a"),
				},
				{
					OmitFrom:     2,
					ContinueFrom: lineNumberEOF,
					Content:      []byte("b"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePatchTextFilePayloads(tt.arg)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr, "error doesn't match expected.")
			} else if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parsePatchTextFilePayloads() = %s, want %s", got, tt.want)
			}
		})
	}
}

func Test_patchTextFileWritePatchedFile(t *testing.T) {
	type arg struct {
		file         []byte
		replacements []patchTextFileReplacement
	}
	tests := []struct {
		name    string
		arg     arg
		wantErr string
		want    []byte
		wantLE  string
	}{
		{
			name: "test no replacements (empty file)",
			arg: arg{
				file:         []byte(""),
				replacements: nil,
			},
			wantLE: "\n",
			want:   nil,
		},
		{
			name: "test no replacements (single line no line ending)",
			arg: arg{
				file:         []byte("l1"),
				replacements: nil,
			},
			wantLE: "\n",
			want:   []byte("l1"),
		},
		{
			name: "test no replacements keeps final line ending (LF)",
			arg: arg{
				file:         []byte("l1\n"),
				replacements: nil,
			},
			wantLE: "\n",
			want:   []byte("l1\n"),
		},
		{
			name: "test no replacements keeps final line ending (CRLF)",
			arg: arg{
				file:         []byte("l1\r\n"),
				replacements: nil,
			},
			wantLE: "\r\n",
			want:   []byte("l1\r\n"),
		},
		{
			name: "test no replacements multiple line endings",
			arg: arg{
				file:         []byte("l1\r\nl2\nl3"),
				replacements: nil,
			},
			wantLE: "\r\n",
			want:   []byte("l1\r\nl2\nl3"),
		},
		{
			name: "test line ending correction with replacements (LF)",
			arg: arg{
				file: []byte("l1\nl2\r\nl3"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     2,
						ContinueFrom: 2,
						Content:      []byte("rl1\nrl2\r\nrl3"),
					},
				},
			},
			wantLE: "\n",
			want:   []byte("l1\nrl1\nrl2\nrl3\nl2\r\nl3"),
		},
		{
			name: "test line ending correction with replacements (CRLF)",
			arg: arg{
				file: []byte("l1\r\nl2\nl3"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     2,
						ContinueFrom: 2,
						Content:      []byte("rl1\nrl2\r\nrl3"),
					},
				},
			},
			wantLE: "\r\n",
			want:   []byte("l1\r\nrl1\r\nrl2\r\nrl3\r\nl2\nl3"),
		},
		{
			name: "test line ending with replacements at eof (file none, replacement none)",
			arg: arg{
				file: []byte("l1\nl2"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     2,
						ContinueFrom: lineNumberEOF,
						Content:      []byte("rl1"),
					},
				},
			},
			wantLE: "\n",
			want:   []byte("l1\nrl1"),
		},
		{
			name: "test line ending with replacements at eof (file none, replacement yes)",
			arg: arg{
				file: []byte("l1\nl2"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     2,
						ContinueFrom: lineNumberEOF,
						Content:      []byte("rl1\r\n"),
					},
				},
			},
			wantLE: "\n",
			want:   []byte("l1\nrl1\n"),
		},
		{
			name: "test line ending with replacements at eof (file yes, replacement none)",
			arg: arg{
				file: []byte("l1\nl2\r\n"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     2,
						ContinueFrom: lineNumberEOF,
						Content:      []byte("rl1"),
					},
				},
			},
			wantLE: "\n",
			want:   []byte("l1\nrl1"),
		},
		{
			name: "test line ending with replacements at eof (file yes, replacement yes)",
			arg: arg{
				file: []byte("l1\nl2\r\n"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     2,
						ContinueFrom: lineNumberEOF,
						Content:      []byte("rl1\r\n"),
					},
				},
			},
			wantLE: "\n",
			want:   []byte("l1\nrl1\n"),
		},
		{
			name: "test final line ending doesn't increase line count",
			arg: arg{
				file: []byte("l1\n"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     3,
						ContinueFrom: 3,
						Content:      []byte("rl1\r\n"),
					},
				},
			},
			wantErr: "Patch action for [3,3) is exceeding end of file with 1 line(s)",
		},
		{
			name: "test replacement out of bounds (start)",
			arg: arg{
				file: []byte("l1"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     3,
						ContinueFrom: lineNumberEOF,
						Content:      []byte("rl1\r\n"),
					},
				},
			},
			wantErr: "Patch action for [3,eof) is exceeding end of file with 1 line(s)",
		},
		{
			name: "test replacement out of bounds (end)",
			arg: arg{
				file: []byte("l1"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     1,
						ContinueFrom: 3,
						Content:      []byte("rl1\r\n"),
					},
				},
			},
			wantErr: "Patch action for [1,3) is exceeding end of file with 1 line(s)",
		},
		{
			name: "test replacement out of bounds (after eof)",
			arg: arg{
				file: []byte("l1"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     1,
						ContinueFrom: lineNumberEOF,
						Content:      []byte("rl1\r\n"),
					},
					{
						OmitFrom:     2,
						ContinueFrom: 3,
						Content:      []byte("rl1\r\n"),
					},
				},
			},
			wantErr: "Patch action for [2,3) is exceeding end of file with 1 line(s)",
		},
		{
			name: "test replacement out of bounds (after last line)",
			arg: arg{
				file: []byte("l1"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     1,
						ContinueFrom: 2,
						Content:      []byte("rl1\r\n"),
					},
					{
						OmitFrom:     3,
						ContinueFrom: 4,
						Content:      []byte("rl1\r\n"),
					},
				},
			},
			wantErr: "Patch action for [3,4) is exceeding end of file with 1 line(s)",
		},
		{
			name: "test overlap before eof (with empty)",
			arg: arg{
				file: []byte(""),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     1,
						ContinueFrom: 3,
						Content:      []byte(""),
					},
					{
						OmitFrom:     2,
						ContinueFrom: 2,
						Content:      []byte(""),
					},
				},
			},
			wantErr: "Patch actions have conflicting ranges [1,3)x[2,2)",
		},
		{
			name: "test overlap before eof (non-empty + unordered)",
			arg: arg{
				file: []byte("l1"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     2,
						ContinueFrom: 3,
						Content:      []byte(""),
					},
					{
						OmitFrom:     1,
						ContinueFrom: 3,
						Content:      []byte(""),
					},
				},
			},
			wantErr: "Patch actions have conflicting ranges [1,3)x[2,3)",
		},
		{
			name: "test overlap before eof (non-empty eof end)",
			arg: arg{
				file: []byte("l1"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     1,
						ContinueFrom: 3,
						Content:      []byte(""),
					},
					{
						OmitFrom:     2,
						ContinueFrom: lineNumberEOF,
						Content:      []byte(""),
					},
				},
			},
			wantErr: "Patch actions have conflicting ranges [1,3)x[2,eof)",
		},
		{
			name: "test overlap after eof (empty)",
			arg: arg{
				file: []byte("l1\nl2\nl3"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     1,
						ContinueFrom: lineNumberEOF,
						Content:      []byte(""),
					},
					{
						OmitFrom:     2,
						ContinueFrom: 2,
						Content:      []byte(""),
					},
				},
			},
			wantErr: "Patch actions have conflicting ranges [1,eof)x[2,2) for file with 3 line(s)",
		},
		{
			name: "test overlap after eof (non-empty + unordered)",
			arg: arg{
				file: []byte("l1\nl2\nl3"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     2,
						ContinueFrom: 3,
						Content:      []byte(""),
					},
					{
						OmitFrom:     1,
						ContinueFrom: lineNumberEOF,
						Content:      []byte(""),
					},
				},
			},
			wantErr: "Patch actions have conflicting ranges [1,eof)x[2,3) for file with 3 line(s)",
		},
		{
			name: "test overlap after eof (none-empty eof end)",
			arg: arg{
				file: []byte("l1\nl2\nl3"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     2,
						ContinueFrom: lineNumberEOF,
						Content:      []byte(""),
					},
					{
						OmitFrom:     1,
						ContinueFrom: lineNumberEOF,
						Content:      []byte(""),
					},
				},
			},
			wantErr: "Patch actions have conflicting ranges [1,eof)x[2,eof) for file with 3 line(s)",
		},
		{
			name: "test insert (empty)",
			arg: arg{
				file: nil,
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     lineNumberEOF,
						ContinueFrom: lineNumberEOF,
						Content:      []byte("rl1\r\nrl2"),
					},
				},
			},
			wantLE: "\n",
			want:   []byte("rl1\nrl2"),
		},
		{
			name: "test insert (start)",
			arg: arg{
				file: []byte("l1"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     1,
						ContinueFrom: 1,
						Content:      []byte("rl1"),
					},
				},
			},
			wantLE: "\n",
			want:   []byte("rl1\nl1"),
		},
		{
			name: "test insert (middle)",
			arg: arg{
				file: []byte("l1\nl2"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     2,
						ContinueFrom: 2,
						Content:      []byte("rl1"),
					},
				},
			},
			wantLE: "\n",
			want:   []byte("l1\nrl1\nl2"),
		},
		{
			name: "test insert (end)",
			arg: arg{
				file: []byte("l1"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     2,
						ContinueFrom: 2,
						Content:      []byte("rl1"),
					},
				},
			},
			wantLE: "\n",
			want:   []byte("l1\nrl1"),
		},
		{
			name: "test insert (eof)",
			arg: arg{
				file: []byte("l1"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     lineNumberEOF,
						ContinueFrom: lineNumberEOF,
						Content:      []byte("rl1"),
					},
				},
			},
			wantLE: "\n",
			want:   []byte("l1\nrl1"),
		},
		{
			name: "test inserts (multiple at start+middle+end(normal+eof))",
			arg: arg{
				file: []byte("l1\nl2\nl3"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     1,
						ContinueFrom: 1,
						Content:      []byte("r1l1\nr1l2"),
					},
					{
						OmitFrom:     1,
						ContinueFrom: 1,
						Content:      []byte("r2l1\nr2l2"),
					},
					{
						OmitFrom:     2,
						ContinueFrom: 2,
						Content:      []byte("r3l1\nr3l2"),
					},
					{
						OmitFrom:     2,
						ContinueFrom: 2,
						Content:      []byte("r4l1\nr4l2"),
					},
					{
						OmitFrom:     4,
						ContinueFrom: 4,
						Content:      []byte("r5l1\nr5l2"),
					},
					{
						OmitFrom:     4,
						ContinueFrom: 4,
						Content:      []byte("r6l1\nr6l2"),
					},
					{
						OmitFrom:     lineNumberEOF,
						ContinueFrom: lineNumberEOF,
						Content:      []byte("r7l1\nr7l2"),
					},
					{
						OmitFrom:     lineNumberEOF,
						ContinueFrom: lineNumberEOF,
						Content:      []byte("r8l1\nr8l2"),
					},
				},
			},
			wantLE: "\n",
			want: []byte(
				"r1l1\nr1l2\nr2l1\nr2l2\nl1\nr3l1\nr3l2\nr4l1\nr4l2\nl2\nl3\nr5l1\nr5l2\nr6l1\nr6l2\nr7l1\nr7l2\nr8l1\nr8l2"),
		},
		{
			name: "test replace (head)",
			arg: arg{
				file: []byte("l1\nl2"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     1,
						ContinueFrom: 2,
						Content:      []byte("rl1"),
					},
				},
			},
			wantLE: "\n",
			want:   []byte("rl1\nl2"),
		},
		{
			name: "test replace (middle)",
			arg: arg{
				file: []byte("l1\nl2\nl3"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     2,
						ContinueFrom: 3,
						Content:      []byte("rl1"),
					},
				},
			},
			wantLE: "\n",
			want:   []byte("l1\nrl1\nl3"),
		},
		{
			name: "test replace (end)",
			arg: arg{
				file: []byte("l1\nl2\nl3"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     3,
						ContinueFrom: 4,
						Content:      []byte("rl1"),
					},
				},
			},
			wantLE: "\n",
			want:   []byte("l1\nl2\nrl1"),
		},
		{
			name: "test replace (eof)",
			arg: arg{
				file: []byte("l1\nl2\nl3"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     3,
						ContinueFrom: lineNumberEOF,
						Content:      []byte("rl1"),
					},
				},
			},
			wantLE: "\n",
			want:   []byte("l1\nl2\nrl1"),
		},
		{
			name: "test replace (1-end)",
			arg: arg{
				file: []byte("l1\nl2\nl3"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     1,
						ContinueFrom: 4,
						Content:      []byte("rl1"),
					},
				},
			},
			wantLE: "\n",
			want:   []byte("rl1"),
		},
		{
			name: "test replace (1-eof)",
			arg: arg{
				file: []byte("l1\nl2\nl3"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     1,
						ContinueFrom: lineNumberEOF,
						Content:      []byte("rl1"),
					},
				},
			},
			wantLE: "\n",
			want:   []byte("rl1"),
		},
		{
			name: "test sorting",
			arg: arg{
				file: []byte("l1\nl2\nl3"),
				replacements: []patchTextFileReplacement{
					{
						OmitFrom:     4,
						ContinueFrom: 4,
						Content:      []byte("r5l1\nr5l2\r\n"),
					},
					{
						OmitFrom:     4,
						ContinueFrom: lineNumberEOF,
						Content:      []byte("r7l1\nr7l2\r\n"),
					},
					{
						OmitFrom:     1,
						ContinueFrom: 1,
						Content:      []byte("r0l1\nr0l2\r\n"),
					},
					{
						OmitFrom:     2,
						ContinueFrom: 4,
						Content:      []byte("r4l1\nr4l2\r\n"),
					},
					{
						OmitFrom:     4,
						ContinueFrom: 4,
						Content:      []byte("r6l1\nr6l2\r\n"),
					},
					{
						OmitFrom:     1,
						ContinueFrom: 2,
						Content:      []byte("r2l1\nr2l2\r\n"),
					},
					{
						OmitFrom:     1,
						ContinueFrom: 1,
						Content:      []byte("r1l1\nr1l2\r\n"),
					},
					{
						OmitFrom:     lineNumberEOF,
						ContinueFrom: lineNumberEOF,
						Content:      []byte("r9l1\nr9l2\r\n"),
					},
					{
						OmitFrom:     4,
						ContinueFrom: lineNumberEOF,
						Content:      []byte("r8l1\nr8l2\r\n"),
					},
					{
						OmitFrom:     2,
						ContinueFrom: 2,
						Content:      []byte("r3l1\nr3l2\r\n"),
					},
					{
						OmitFrom:     lineNumberEOF,
						ContinueFrom: lineNumberEOF,
						Content:      []byte("r10l1\nr10l2\r\n"),
					},
				},
			},
			wantLE: "\n",
			want: []byte("r0l1\nr0l2\nr1l1\nr1l2\nr2l1\nr2l2\nr3l1\nr3l2\nr4l1\nr4l2\nr5l1\nr5l2\nr6l1\nr6l2\nr7l1\nr7l2" +
				"\nr8l1\nr8l2\nr9l1\nr9l2\nr10l1\nr10l2\n"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner, le, err := parser.ReadTextFile(bytes.NewReader(tt.arg.file), nil)
			require.NoError(t, err, "failed to read input file")

			writer := bytes.Buffer{}

			err = patchTextFileWritePatchedFile(scanner, tt.arg.replacements, le, &writer)
			got := writer.Bytes()
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr, "error doesn't match expected.")
			} else {
				assert.Equal(t, tt.wantLE, le, "line ending doesn't match")
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("patchTextFileWritePatchedFile() = %q, want %q", string(got), string(tt.want))
				}
			}
		})
	}
}
