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
	"bufio"
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_scannerWithPeekSmoke(t *testing.T) {
	scanner := NewScannerWithPeek(
		bytes.NewReader([]byte("l1\nl2")),
		bufio.ScanLines,
	)

	out := scanner.Peek()
	require.True(t, out)
	require.NoError(t, scanner.Err())
	require.Equal(t, "l1", string(scanner.Bytes()))

	out = scanner.Scan()
	require.True(t, out)
	require.NoError(t, scanner.Err())
	require.Equal(t, "l1", string(scanner.Bytes()))

	out = scanner.Scan()
	require.True(t, out)
	require.NoError(t, scanner.Err())
	require.Equal(t, "l2", scanner.Text())

	out = scanner.Scan()
	require.False(t, out)
	require.NoError(t, scanner.Err())
	require.Nil(t, scanner.Bytes())
}

func Test_scannerWithPeekDualPeek(t *testing.T) {
	scanner := NewScannerWithPeek(
		bytes.NewReader([]byte("l1\nl2")),
		bufio.ScanLines,
	)

	out := scanner.Peek()
	require.True(t, out)
	require.NoError(t, scanner.Err())
	require.Equal(t, "l1", string(scanner.Bytes()))

	out = scanner.Peek()
	require.False(t, out)
	require.ErrorIs(t, scanner.Err(), ErrPeekedMoreThanOnce)
	require.Nil(t, scanner.Bytes())
}
