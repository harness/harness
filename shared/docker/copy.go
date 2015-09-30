package docker

import (
	"encoding/binary"
	"errors"
	"io"
)

const (
	StdWriterPrefixLen = 8
	StdWriterFdIndex   = 0
	StdWriterSizeIndex = 4
)

type StdType [StdWriterPrefixLen]byte

var (
	Stdin  StdType = StdType{0: 0}
	Stdout StdType = StdType{0: 1}
	Stderr StdType = StdType{0: 2}
)

type StdWriter struct {
	io.Writer
	prefix  StdType
	sizeBuf []byte
}

var ErrInvalidStdHeader = errors.New("Unrecognized input header")

// StdCopy is a modified version of io.Copy.
//
// StdCopy will demultiplex `src`, assuming that it contains two streams,
// previously multiplexed together using a StdWriter instance.
// As it reads from `src`, StdCopy will write to `dstout` and `dsterr`.
//
// StdCopy will read until it hits EOF on `src`. It will then return a nil error.
// In other words: if `err` is non nil, it indicates a real underlying error.
//
// `written` will hold the total number of bytes written to `dstout` and `dsterr`.
func StdCopy(dstout, dsterr io.Writer, src io.Reader) (written int64, err error) {
	var (
		buf       = make([]byte, 32*1024+StdWriterPrefixLen+1)
		bufLen    = len(buf)
		nr, nw    int
		er, ew    error
		out       io.Writer
		frameSize int
	)

	for {
		// Make sure we have at least a full header
		for nr < StdWriterPrefixLen {
			var nr2 int
			nr2, er = src.Read(buf[nr:])
			nr += nr2
			if er == io.EOF {
				if nr < StdWriterPrefixLen {
					return written, nil
				}
				break
			}
			if er != nil {
				return 0, er
			}
		}

		// Check the first byte to know where to write
		switch buf[StdWriterFdIndex] {
		case 0:
			fallthrough
		case 1:
			// Write on stdout
			out = dstout
		case 2:
			// Write on stderr
			out = dsterr
		default:
			return 0, ErrInvalidStdHeader
		}

		// Retrieve the size of the frame
		frameSize = int(binary.BigEndian.Uint32(buf[StdWriterSizeIndex : StdWriterSizeIndex+4]))

		// Check if the buffer is big enough to read the frame.
		// Extend it if necessary.
		if frameSize+StdWriterPrefixLen > bufLen {
			buf = append(buf, make([]byte, frameSize+StdWriterPrefixLen-bufLen+1)...)
			bufLen = len(buf)
		}

		// While the amount of bytes read is less than the size of the frame + header, we keep reading
		for nr < frameSize+StdWriterPrefixLen {
			var nr2 int
			nr2, er = src.Read(buf[nr:])
			nr += nr2
			if er == io.EOF {
				if nr < frameSize+StdWriterPrefixLen {
					return written, nil
				}
				break
			}
			if er != nil {
				return 0, er
			}
		}

		// Write the retrieved frame (without header)
		nw, ew = out.Write(buf[StdWriterPrefixLen : frameSize+StdWriterPrefixLen])
		if ew != nil {
			return 0, ew
		}
		// If the frame has not been fully written: error
		if nw != frameSize {
			return 0, io.ErrShortWrite
		}
		written += int64(nw)

		// Move the rest of the buffer to the beginning
		copy(buf, buf[frameSize+StdWriterPrefixLen:])
		// Move the index
		nr -= frameSize + StdWriterPrefixLen
	}
}
