// hello.go ported for appengine
//
// this differs from the standard hello.go example in two ways: appengine
// already provides an http server for you, obviating the need for the
// ListenAndServe call (with associated logging), and the package must not be
// called main (appengine reserves package 'main' for the underlying program).

package patexample

import (
    "io"
    "net/http"
    "github.com/bmizerany/pat"
)

// hello world, the web server
func HelloServer(w http.ResponseWriter, req *http.Request) {
    io.WriteString(w, "hello, "+req.URL.Query().Get(":name")+"!\n")
}

func init() {
    m := pat.New()
    m.Get("/hello/:name", http.HandlerFunc(HelloServer))

    // Register this pat with the default serve mux so that other packages
    // may also be exported. (i.e. /debug/pprof/*)
    http.Handle("/", m)
}
