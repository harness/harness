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

package hash

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	byte1 = []byte{1}
	byte2 = []byte{2}
)

func TestSourceFromChannel_blockingChannel(t *testing.T) {
	nextChan := make(chan SourceNext)

	ctx, cncl := context.WithTimeout(context.Background(), 1*time.Second)
	defer cncl()

	source := SourceFromChannel(ctx, nextChan)

	go func() {
		defer close(nextChan)

		select {
		case nextChan <- SourceNext{Data: byte1}:
		case <-ctx.Done():
			require.Fail(t, "writing data to source chan timed out")
		}
	}()

	next, err := source.Next()
	require.NoError(t, err, "no error expected on first call to next")
	require.Equal(t, byte1, next)

	_, err = source.Next()
	require.ErrorIs(t, err, io.EOF, "EOF expected after first element was read")
}

func TestSourceFromChannel_contextCanceled(t *testing.T) {
	nextChan := make(chan SourceNext)

	ctx, cncl := context.WithTimeout(context.Background(), 1*time.Second)
	cncl()

	source := SourceFromChannel(ctx, nextChan)
	_, err := source.Next()
	require.ErrorIs(t, err, context.Canceled, "Canceled error expected")
}

func TestSourceFromChannel_sourceChannelDrainedOnClosing(t *testing.T) {
	nextChan := make(chan SourceNext, 1)

	ctx, cncl := context.WithTimeout(context.Background(), 1*time.Second)
	defer cncl()

	source := SourceFromChannel(ctx, nextChan)

	nextChan <- SourceNext{Data: byte1}
	close(nextChan)

	next, err := source.Next()
	require.NoError(t, err, "no error expected on first call to next")
	require.Equal(t, byte1, next)

	_, err = source.Next()
	require.ErrorIs(t, err, io.EOF, "EOF expected after first element was read")
}

func TestSourceFromChannel_errorReturnedOnError(t *testing.T) {
	nextChan := make(chan SourceNext, 1)

	ctx, cncl := context.WithTimeout(context.Background(), 1*time.Second)
	defer cncl()

	source := SourceFromChannel(ctx, nextChan)

	nextChan <- SourceNext{
		Data: byte1,
		Err:  io.ErrClosedPipe,
	}

	next, err := source.Next()
	require.ErrorIs(t, err, io.ErrClosedPipe, "ErrClosedPipe expected")
	require.Equal(t, byte1, next)
}

func TestSourceFromChannel_fullChannel(t *testing.T) {
	nextChan := make(chan SourceNext, 1)

	ctx, cncl := context.WithTimeout(context.Background(), 1*time.Second)
	defer cncl()

	source := SourceFromChannel(ctx, nextChan)

	nextChan <- SourceNext{Data: byte1}

	go func() {
		defer close(nextChan)

		select {
		case nextChan <- SourceNext{Data: byte2}:
		case <-ctx.Done():
			require.Fail(t, "writing data to source chan timed out")
		}
	}()

	next, err := source.Next()
	require.NoError(t, err, "no error expected on first call to next")
	require.Equal(t, byte1, next)

	next, err = source.Next()
	require.NoError(t, err, "no error expected on second call to next")
	require.Equal(t, byte2, next)

	_, err = source.Next()
	require.ErrorIs(t, err, io.EOF, "EOF expected after two elements were read")
}
