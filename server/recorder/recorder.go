package recorder

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
)

type ResponseRecorder struct {
	*httptest.ResponseRecorder
}

func NewResponseRecorder() *ResponseRecorder {
	return &ResponseRecorder{httptest.NewRecorder()}
}

func (rr *ResponseRecorder) reset() {
	rr.ResponseRecorder = httptest.NewRecorder()
}

func (rr *ResponseRecorder) CloseNotify() <-chan bool {
	return http.ResponseWriter(rr).(http.CloseNotifier).CloseNotify()
}

func (rr *ResponseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.ResponseWriter(rr).(http.Hijacker).Hijack()
}

func (rr *ResponseRecorder) Size() int       { return rr.Body.Len() }
func (rr *ResponseRecorder) Status() int     { return rr.Code }
func (rr *ResponseRecorder) WriteHeaderNow() {}
func (rr *ResponseRecorder) Written() bool   { return rr.Code != 0 }
