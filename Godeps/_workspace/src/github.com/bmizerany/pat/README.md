# pat (formerly pat.go) - A Sinatra style pattern muxer for Go's net/http library

## INSTALL

	$ go get github.com/bmizerany/pat

## USE

	package main

	import (
		"io"
		"net/http"
		"github.com/bmizerany/pat"
		"log"
	)

	// hello world, the web server
	func HelloServer(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "hello, "+req.URL.Query().Get(":name")+"!\n")
	}

	func main() {
		m := pat.New()
		m.Get("/hello/:name", http.HandlerFunc(HelloServer))

		// Register this pat with the default serve mux so that other packages
		// may also be exported. (i.e. /debug/pprof/*)
		http.Handle("/", m)
		err := http.ListenAndServe(":12345", nil)
		if err != nil {
			log.Fatal("ListenAndServe: ", err)
		}
	}
	
It's that simple.

For more information, see:
http://gopkgdoc.appspot.com/pkg/github.com/bmizerany/pat

## CONTRIBUTORS

* Keith Rarick (@krarick) - github.com/kr
* Blake Mizerany (@bmizerany) - github.com/bmizerany
* Evan Shaw
* George Rogers

## LICENSE

Copyright (C) 2012 by Keith Rarick, Blake Mizerany

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE. 
