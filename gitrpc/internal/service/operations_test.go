// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func Test_parsePayload(t *testing.T) {
	s := "this is the content of the file"
	filename := "file.txt"
	content := &bytes.Buffer{}
	type args struct {
		payload io.Reader
		content io.Writer
	}
	tests := []struct {
		name        string
		args        args
		want        string
		wantContent []byte
		wantErr     bool
	}{
		{
			name: "no content",
			args: args{
				payload: strings.NewReader(""),
				content: content,
			},
		},
		{
			name: "sample content",
			args: args{
				payload: strings.NewReader(s),
				content: content,
			},
			wantContent: []byte(s),
		},
		{
			name: "file name test",
			args: args{
				payload: strings.NewReader(filePrefix + filename),
				content: content,
			},
			want:        filename,
			wantContent: []byte{},
		},
		{
			name: "file name with new line",
			args: args{
				payload: strings.NewReader(filePrefix + filename + "\n"),
				content: content,
			},
			want:        filename,
			wantContent: []byte{},
		},
		{
			name: "content test",
			args: args{
				payload: strings.NewReader(filePrefix + filename + "\n" + s),
				content: content,
			},
			want:        filename,
			wantContent: []byte(s),
		},
		{
			name: "content test with empty line at the top",
			args: args{
				payload: strings.NewReader(filePrefix + filename + "\n\n" + s),
				content: content,
			},
			want:        filename,
			wantContent: []byte("\n" + s),
		},
		{
			name: "content test with double empty line at the top",
			args: args{
				payload: strings.NewReader(filePrefix + filename + "\n\n\n" + s),
				content: content,
			},
			want:        filename,
			wantContent: []byte("\n\n" + s),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePayload(tt.args.payload, tt.args.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parsePayload() got filename = %v, want %v",
					got, tt.want)
			}
			if !bytes.Equal(content.Bytes(), tt.wantContent) {
				t.Errorf("parsePayload() got content = %v, want %v",
					content.Bytes(), tt.wantContent)
			}
			content.Reset()
		})
	}
}
