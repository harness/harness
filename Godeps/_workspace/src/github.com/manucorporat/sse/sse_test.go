// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package sse

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeOnlyData(t *testing.T) {
	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Data: "junk\n\njk\nid:fake",
	})
	assert.NoError(t, err)
	assert.Equal(t, w.String(), "data: junk\\n\\njk\\nid:fake\n\n")
}

func TestEncodeWithEvent(t *testing.T) {
	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Event: "t\n:<>\r\test",
		Data:  "junk\n\njk\nid:fake",
	})
	assert.NoError(t, err)
	assert.Equal(t, w.String(), "event: t\\n:<>\\r\test\ndata: junk\\n\\njk\\nid:fake\n\n")
}

func TestEncodeWithId(t *testing.T) {
	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Id:   "t\n:<>\r\test",
		Data: "junk\n\njk\nid:fa\rke",
	})
	assert.NoError(t, err)
	assert.Equal(t, w.String(), "id: t\\n:<>\\r\test\ndata: junk\\n\\njk\\nid:fa\\rke\n\n")
}

func TestEncodeWithRetry(t *testing.T) {
	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Retry: 11,
		Data:  "junk\n\njk\nid:fake\n",
	})
	assert.NoError(t, err)
	assert.Equal(t, w.String(), "retry: 11\ndata: junk\\n\\njk\\nid:fake\\n\n\n")
}

func TestEncodeWithEverything(t *testing.T) {
	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Event: "abc",
		Id:    "12345",
		Retry: 10,
		Data:  "some data",
	})
	assert.NoError(t, err)
	assert.Equal(t, w.String(), "id: 12345\nevent: abc\nretry: 10\ndata: some data\n\n")
}

func TestEncodeMap(t *testing.T) {
	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Event: "a map",
		Data: map[string]interface{}{
			"foo": "b\n\rar",
			"bar": "id: 2",
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, w.String(), "event: a map\ndata: {\"bar\":\"id: 2\",\"foo\":\"b\\n\\rar\"}\n\n")
}

func TestEncodeSlice(t *testing.T) {
	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Event: "a slice",
		Data:  []interface{}{1, "text", map[string]interface{}{"foo": "bar"}},
	})
	assert.NoError(t, err)
	assert.Equal(t, w.String(), "event: a slice\ndata: [1,\"text\",{\"foo\":\"bar\"}]\n\n")
}

func TestEncodeStruct(t *testing.T) {
	myStruct := struct {
		A int
		B string `json:"value"`
	}{1, "number"}

	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Event: "a struct",
		Data:  myStruct,
	})
	assert.NoError(t, err)
	assert.Equal(t, w.String(), "event: a struct\ndata: {\"A\":1,\"value\":\"number\"}\n\n")

	w.Reset()
	err = Encode(w, Event{
		Event: "a struct",
		Data:  &myStruct,
	})
	assert.NoError(t, err)
	assert.Equal(t, w.String(), "event: a struct\ndata: {\"A\":1,\"value\":\"number\"}\n\n")
}

func TestEncodeInteger(t *testing.T) {
	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Event: "an integer",
		Data:  1,
	})
	assert.NoError(t, err)
	assert.Equal(t, w.String(), "event: an integer\ndata: 1\n\n")
}

func TestEncodeFloat(t *testing.T) {
	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Event: "Float",
		Data:  1.5,
	})
	assert.NoError(t, err)
	assert.Equal(t, w.String(), "event: Float\ndata: 1.5\n\n")
}

func TestEncodeStream(t *testing.T) {
	w := new(bytes.Buffer)

	Encode(w, Event{
		Event: "float",
		Data:  1.5,
	})

	Encode(w, Event{
		Id:   "123",
		Data: map[string]interface{}{"foo": "bar", "bar": "foo"},
	})

	Encode(w, Event{
		Id:    "124",
		Event: "chat",
		Data:  "hi! dude",
	})
	assert.Equal(t, w.String(), "event: float\ndata: 1.5\n\nid: 123\ndata: {\"bar\":\"foo\",\"foo\":\"bar\"}\n\nid: 124\nevent: chat\ndata: hi! dude\n\n")
}

func TestRenderSSE(t *testing.T) {
	w := httptest.NewRecorder()

	err := (Event{
		Event: "msg",
		Data:  "hi! how are you?",
	}).Render(w)

	assert.NoError(t, err)
	assert.Equal(t, w.Body.String(), "event: msg\ndata: hi! how are you?\n\n")
	assert.Equal(t, w.Header().Get("Content-Type"), "text/event-stream")
	assert.Equal(t, w.Header().Get("Cache-Control"), "no-cache")
}

func BenchmarkResponseWriter(b *testing.B) {
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		(Event{
			Event: "new_message",
			Data:  "hi! how are you? I am fine. this is a long stupid message!!!",
		}).Render(w)
	}
}

func BenchmarkFullSSE(b *testing.B) {
	buf := new(bytes.Buffer)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		Encode(buf, Event{
			Event: "new_message",
			Id:    "13435",
			Retry: 10,
			Data:  "hi! how are you? I am fine. this is a long stupid message!!!",
		})
		buf.Reset()
	}
}

func BenchmarkNoRetrySSE(b *testing.B) {
	buf := new(bytes.Buffer)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		Encode(buf, Event{
			Event: "new_message",
			Id:    "13435",
			Data:  "hi! how are you? I am fine. this is a long stupid message!!!",
		})
		buf.Reset()
	}
}

func BenchmarkSimpleSSE(b *testing.B) {
	buf := new(bytes.Buffer)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		Encode(buf, Event{
			Event: "new_message",
			Data:  "hi! how are you? I am fine. this is a long stupid message!!!",
		})
		buf.Reset()
	}
}
