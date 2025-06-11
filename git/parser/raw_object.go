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

package parser

import (
	"bytes"
	"fmt"
	"regexp"
)

var (
	reSigBegin = regexp.MustCompile(`^-----BEGIN (.+)-----\n`)
	reSigEnd   = regexp.MustCompile(`^-----END (.+)-----\n`)
)

type ObjectRaw struct {
	Headers       []ObjectHeader
	Message       string
	SignedContent []byte
	Signature     []byte
	SignatureType string
}

type ObjectHeader struct {
	Type  string
	Value string
}

func Object(raw []byte) (ObjectRaw, error) {
	var (
		headers       []ObjectHeader
		message       bytes.Buffer
		signedContent bytes.Buffer
		sigBuffer     bytes.Buffer
		sig           []byte
		sigType       string
	)

	headers = make([]ObjectHeader, 0, 4)

	const (
		stateHead = iota
		stateBody
		stateBodySig
	)

	state := stateHead

	for {
		if len(raw) == 0 {
			break
		}

		if state == stateHead {
			headerType, headerContent, advance, err := getHeader(raw)
			if err != nil {
				return ObjectRaw{}, err
			}

			switch headerType {
			case "":
				signedContent.Write(raw[:advance])
				state = stateBody
			case "gpgsig":
				sig = headerContent.Bytes()

				data := reSigBegin.FindSubmatch(sig)
				if len(data) == 0 {
					return ObjectRaw{}, fmt.Errorf("invalid signature header: %s", data)
				}

				sigType = string(data[1])
			default:
				signedContent.Write(raw[:advance])

				// Headers added with the trailing EOL character. This is important for some headers (mergetag).
				headerValue := headerContent.String()

				headers = append(headers, ObjectHeader{
					Type:  headerType,
					Value: headerValue,
				})
			}

			raw = raw[advance:]
			continue
		}

		var line []byte

		idxEOL := bytes.IndexByte(raw, '\n')
		if idxEOL == -1 {
			line = raw
			raw = nil
		} else {
			line = raw[:idxEOL+1] // line includes EOL
			raw = raw[idxEOL+1:]
		}

		if state == stateBodySig {
			sigBuffer.Write(line)
			if reSigEnd.Match(line) {
				sig = sigBuffer.Bytes()
				state = stateBody
			}
			continue
		}

		data := reSigBegin.FindSubmatch(line)
		if len(data) > 0 {
			sigBuffer.Write(line)
			sigType = string(data[1])
			state = stateBodySig
		} else {
			signedContent.Write(line)
			message.Write(line)
		}
	}

	var signedContentBytes []byte
	if sigType != "" {
		signedContentBytes = signedContent.Bytes()
	}

	return ObjectRaw{
		Headers:       headers,
		Message:       message.String(),
		SignedContent: signedContentBytes,
		Signature:     sig,
		SignatureType: sigType,
	}, nil
}

func getHeader(raw []byte) (headerType string, headerContent *bytes.Buffer, advance int, err error) {
	headerContent = bytes.NewBuffer(nil)

	for {
		idxEOL := bytes.IndexByte(raw, '\n')
		if idxEOL < 0 {
			return "", nil, 0, fmt.Errorf("header line must end with EOL character: %s", raw)
		}

		lineLen := idxEOL + 1
		line := raw[:lineLen] // line includes EOL
		raw = raw[lineLen:]

		if advance == 0 {
			if len(line) == 1 {
				// empty line means no header
				return "", nil, 1, nil
			}

			idxSpace := bytes.IndexByte(line, ' ') // expected header form is "<header-type><space><header_value>"
			if idxSpace <= 0 {
				return "", nil, 0, fmt.Errorf("malformed header: %s", line[:len(line)-1])
			}

			headerType = string(line[:idxSpace])
			headerContent.Write(line[idxSpace+1:])
		} else {
			headerContent.Write(line[1:]) // add the line without the space prefix
		}

		advance += lineLen

		// peak at next line to find if it's a multiline header (i.e. a signature) - if the next line starts with space
		hasMoreLines := len(raw) > 0 && raw[0] == ' '
		if !hasMoreLines {
			return headerType, headerContent, advance, nil
		}
	}
}
