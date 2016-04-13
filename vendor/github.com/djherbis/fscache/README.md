fscache 
==========

[![GoDoc](https://godoc.org/github.com/djherbis/fscache?status.svg)](https://godoc.org/github.com/djherbis/fscache)
[![Release](https://img.shields.io/github/release/djherbis/fscache.svg)](https://github.com/djherbis/fscache/releases/latest)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg)](LICENSE.txt)
[![Build Status](https://travis-ci.org/djherbis/fscache.svg?branch=master)](https://travis-ci.org/djherbis/fscache)
[![Coverage Status](https://coveralls.io/repos/djherbis/fscache/badge.svg?branch=master)](https://coveralls.io/r/djherbis/fscache?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/djherbis/fscache)](https://goreportcard.com/report/github.com/djherbis/fscache)

Usage
------------
Streaming File Cache for #golang

fscache allows multiple readers to read from a cache while its being written to. [blog post](https://djherbis.github.io/post/fscache/)

Using the Cache directly:

```go
package main

import (
	"io"
	"log"
	"os"
	"time"

	"gopkg.in/djherbis/fscache.v0"
)

func main() {

	// create the cache, keys expire after 1 hour.
	c, err := fscache.New("./cache", 0755, time.Hour)
	if err != nil {
		log.Fatal(err.Error())
	}
	
	// wipe the cache when done
	defer c.Clean()

	// Get() and it's streams can be called concurrently but just for example:
	for i := 0; i < 3; i++ {
		r, w, err := c.Get("stream")
		if err != nil {
			log.Fatal(err.Error())
		}

		if w != nil { // a new stream, write to it.
			go func(){
				w.Write([]byte("hello world\n"))
				w.Close()
			}()
		}

		// the stream has started, read from it
		io.Copy(os.Stdout, r)
		r.Close()
	}
}
```

A Caching Middle-ware:

```go
package main

import(
	"net/http"
	"time"

	"gopkg.in/djherbis/fscache.v0"
)

func main(){
	c, err := fscache.New("./cache", 0700, 0)
	if err != nil {
		log.Fatal(err.Error())
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%v: %s", time.Now(), "hello world")
	}

	http.ListenAndServe(":8080", fscache.Handler(c, http.HandlerFunc(handler)))
}
```

Installation
------------
```sh
go get gopkg.in/djherbis/fscache.v0
```
