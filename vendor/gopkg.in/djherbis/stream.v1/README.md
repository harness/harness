stream 
==========

[![GoDoc](https://godoc.org/github.com/djherbis/stream?status.svg)](https://godoc.org/github.com/djherbis/stream)
[![Release](https://img.shields.io/github/release/djherbis/stream.svg)](https://github.com/djherbis/stream/releases/latest)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg)](LICENSE.txt)
[![Build Status](https://travis-ci.org/djherbis/stream.svg?branch=master)](https://travis-ci.org/djherbis/stream)
[![Coverage Status](https://coveralls.io/repos/djherbis/stream/badge.svg?branch=master)](https://coveralls.io/r/djherbis/stream?branch=master)

Usage
------------

Write and Read concurrently, and independently.

To explain further, if you need to write to multiple places you can use io.MultiWriter,
if you need multiple Readers on something you can use io.TeeReader. If you want concurrency you can use io.Pipe(). 

However all of these methods "tie" each Read/Write together, your readers can't read from different places in the stream, each write must be distributed to all readers in sequence. 

This package provides a way for multiple Readers to read off the same Writer, without waiting for the others. This is done by writing to a "File" interface which buffers the input so it can be read at any time from many independent readers. Readers can even be created while writing or after the stream is closed. They will all see a consistent view of the stream and will block until the section of the stream they request is written, all while being unaffected by the actions of the other readers.

The use case for this stems from my other project djherbis/fscache. I needed a byte caching mechanism which allowed many independent clients to have access to the data while it was being written, rather than re-generating the byte stream for each of them or waiting for a complete copy of the stream which could be stored and then re-used.

```go
import(
	"io"
	"log"
	"os"
	"time"

	"github.com/djherbis/stream"
)

func main(){
	w, err := stream.New("mystream")
	if err != nil {
		log.Fatal(err)
	}

	go func(){
		io.WriteString(w, "Hello World!")
		<-time.After(time.Second)
		io.WriteString(w, "Streaming updates...")
		w.Close()
	}()

	waitForReader := make(chan struct{})
	go func(){
		// Read from the stream
		r, err := w.NextReader()
		if err != nil {
			log.Fatal(err)
		}
		io.Copy(os.Stdout, r) // Hello World! (1 second) Streaming updates...
		r.Close()
		close(waitForReader)
	}()

  // Full copy of the stream!
	r, err := w.NextReader() 
	if err != nil {
		log.Fatal(err)
	}
	io.Copy(os.Stdout, r) // Hello World! (1 second) Streaming updates...

	// r supports io.ReaderAt too.
	p := make([]byte, 4)
	r.ReadAt(p, 1) // Read "ello" into p

	r.Close()

	<-waitForReader // don't leave main before go-routine finishes
}
```

Installation
------------
```sh
go get github.com/djherbis/stream
```
