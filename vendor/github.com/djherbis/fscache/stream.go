package fscache

import (
	"encoding/json"
	"errors"
	"io"
)

type decoder interface {
	Decode(interface{}) error
}

type encoder interface {
	Encode(interface{}) error
}

type pktReader struct {
	dec decoder
}

type pktWriter struct {
	enc encoder
}

type packet struct {
	Err  int
	Data []byte
}

const eof = 1

func (t *pktReader) ReadAt(p []byte, off int64) (n int, err error) {
	// TODO not implemented
	return 0, errors.New("not implemented")
}

func (t *pktReader) Read(p []byte) (int, error) {
	var pkt packet
	err := t.dec.Decode(&pkt)
	if err != nil {
		return 0, err
	}
	if pkt.Err == eof {
		return 0, io.EOF
	}
	return copy(p, pkt.Data), nil
}

func (t *pktReader) Close() error {
	return nil
}

func (t *pktWriter) Write(p []byte) (int, error) {
	pkt := packet{Data: p}
	err := t.enc.Encode(pkt)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (t *pktWriter) Close() error {
	return t.enc.Encode(packet{Err: eof})
}

func newEncoder(w io.Writer) io.WriteCloser {
	return &pktWriter{enc: json.NewEncoder(w)}
}

func newDecoder(r io.Reader) ReadAtCloser {
	return &pktReader{dec: json.NewDecoder(r)}
}
