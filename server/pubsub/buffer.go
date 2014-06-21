package pubsub

import (
	"bytes"
)

type Buffer struct {
	buf     bytes.Buffer
	channel *Channel
}

func NewBuffer(channel *Channel) *Buffer {
	return &Buffer{
		channel: channel,
	}
}

func (b *Buffer) Write(p []byte) (n int, err error) {
	n, err = b.buf.Write(p)
	b.channel.Publish(p)
	return
}

func (b *Buffer) WriteString(s string) (n int, err error) {
	return b.Write([]byte(s))
}

func (b *Buffer) Bytes() []byte {
	return b.buf.Bytes()
}
