package gogitlab

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
)

func Stub(filename string) (*httptest.Server, *Gitlab) {
	stub, _ := ioutil.ReadFile(filename)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(stub))
	}))
	gitlab := NewGitlab(ts.URL, "", "")
	return ts, gitlab
}
