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
	"testing"

	"github.com/gotidy/ptr"
	"github.com/stretchr/testify/assert"
)

func Test_readTextFileEmpty(t *testing.T) {
	scanner, le, err := ReadTextFile(bytes.NewReader(nil), nil)
	assert.NoError(t, err)
	assert.Equal(t, "\n", le)

	ok := scanner.Scan()
	assert.False(t, ok)
	assert.NoError(t, scanner.Err())
}

func Test_readTextFileFirstLineNotUTF8(t *testing.T) {
	scanner, _, err := ReadTextFile(bytes.NewReader([]byte{0xFF, 0xFF}), nil)

	// method itself doesn't return an error, only the scanning fails for utf8.
	assert.NotNil(t, scanner)
	assert.NoError(t, err)

	ok := scanner.Scan()
	assert.False(t, ok)
	assert.ErrorIs(t, scanner.Err(), ErrBinaryFile)
}

func Test_readTextFileNoLineEnding(t *testing.T) {
	scanner, le, err := ReadTextFile(bytes.NewReader([]byte("abc")), nil)
	assert.NoError(t, err)
	assert.Equal(t, "\n", le)

	ok := scanner.Scan()
	assert.True(t, ok)
	assert.NoError(t, scanner.Err())
	assert.Equal(t, "abc", scanner.Text())

	ok = scanner.Scan()
	assert.False(t, ok)
	assert.NoError(t, scanner.Err())
	assert.Nil(t, scanner.Bytes())
}

func Test_readTextFileLineEndingLF(t *testing.T) {
	scanner, le, err := ReadTextFile(bytes.NewReader([]byte("abc\n")), nil)
	assert.NoError(t, err)
	assert.Equal(t, "\n", le)

	ok := scanner.Scan()
	assert.True(t, ok)
	assert.NoError(t, scanner.Err())
	assert.Equal(t, "abc\n", scanner.Text())

	ok = scanner.Scan()
	assert.False(t, ok)
	assert.NoError(t, scanner.Err())
	assert.Nil(t, scanner.Bytes())
}

func Test_readTextFileLineEndingCRLF(t *testing.T) {
	scanner, le, err := ReadTextFile(bytes.NewReader([]byte("abc\r\n")), nil)
	assert.NoError(t, err)
	assert.Equal(t, "\r\n", le)

	ok := scanner.Scan()
	assert.True(t, ok)
	assert.NoError(t, scanner.Err())
	assert.Equal(t, "abc\r\n", scanner.Text())

	ok = scanner.Scan()
	assert.False(t, ok)
	assert.NoError(t, scanner.Err())
	assert.Nil(t, scanner.Bytes())
}

func Test_readTextFileLineEndingMultiple(t *testing.T) {
	scanner, le, err := ReadTextFile(bytes.NewReader([]byte("abc\r\nd\n")), nil)
	assert.NoError(t, err)
	assert.Equal(t, "\r\n", le)

	ok := scanner.Scan()
	assert.True(t, ok)
	assert.NoError(t, scanner.Err())
	assert.Equal(t, "abc\r\n", scanner.Text())

	ok = scanner.Scan()
	assert.True(t, ok)
	assert.NoError(t, scanner.Err())
	assert.Equal(t, "d\n", scanner.Text())

	ok = scanner.Scan()
	assert.False(t, ok)
	assert.NoError(t, scanner.Err())
	assert.Nil(t, scanner.Bytes())
}

func Test_readTextFileLineEndingReplacementEmpty(t *testing.T) {
	scanner, le, err := ReadTextFile(bytes.NewReader([]byte("abc\r\n")), ptr.Of(""))
	assert.NoError(t, err)
	assert.Equal(t, "\r\n", le)

	ok := scanner.Scan()
	assert.True(t, ok)
	assert.NoError(t, scanner.Err())
	assert.Equal(t, "abc", scanner.Text())
}

func Test_readTextFileLineEndingReplacement(t *testing.T) {
	scanner, le, err := ReadTextFile(bytes.NewReader([]byte("abc\r\nd")), ptr.Of("\n"))
	assert.NoError(t, err)
	assert.Equal(t, "\r\n", le)

	ok := scanner.Scan()
	assert.True(t, ok)
	assert.NoError(t, scanner.Err())
	assert.Equal(t, "abc\n", scanner.Text())

	ok = scanner.Scan()
	assert.True(t, ok)
	assert.NoError(t, scanner.Err())
	assert.Equal(t, "d", scanner.Text())

	ok = scanner.Scan()
	assert.False(t, ok)
	assert.NoError(t, scanner.Err())
	assert.Nil(t, scanner.Bytes())
}
