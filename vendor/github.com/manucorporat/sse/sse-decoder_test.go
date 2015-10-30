// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package sse

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeSingle1(t *testing.T) {
	events, err := Decode(bytes.NewBufferString(
		`data: this is a text
event: message
fake:
id: 123456789010
: we can append data
: and multiple comments should not break it
data: a very nice one`))

	assert.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, events[0].Event, "message")
	assert.Equal(t, events[0].Id, "123456789010")
}

func TestDecodeSingle2(t *testing.T) {
	events, err := Decode(bytes.NewBufferString(
		`: starting with a comment
fake:

data:this is a \ntext
event:a message\n\n
fake
:and multiple comments\n should not break it\n\n
id:1234567890\n10
:we can append data
data:a very nice one\n!


`))
	assert.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, events[0].Event, "a message\\n\\n")
	assert.Equal(t, events[0].Id, "1234567890\\n10")
}

func TestDecodeSingle3(t *testing.T) {
	events, err := Decode(bytes.NewBufferString(
		`
id:123456ABCabc789010
event: message123
: we can append data
data:this is a text
data: a very nice one
data:
data
: ending with a comment`))

	assert.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, events[0].Event, "message123")
	assert.Equal(t, events[0].Id, "123456ABCabc789010")
}

func TestDecodeMulti1(t *testing.T) {
	events, err := Decode(bytes.NewBufferString(
		`
id:
event: weird event
data:this is a text
:data: this should NOT APER
data:  second line

: a comment
event: message
id:123
data:this is a text
:data: this should NOT APER
data:  second line


: a comment
event: message
id:123
data:this is a text
data:  second line

:hola

data

event:

id`))
	assert.NoError(t, err)
	assert.Len(t, events, 3)
	assert.Equal(t, events[0].Event, "weird event")
	assert.Equal(t, events[0].Id, "")
}

func TestDecodeW3C(t *testing.T) {
	events, err := Decode(bytes.NewBufferString(
		`data

data
data

data:
`))
	assert.NoError(t, err)
	assert.Len(t, events, 1)
}
