package raven

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"
)

type testcase struct {
	request *http.Request
	*Http
}

func newBaseRequest() *http.Request {
	u, _ := url.Parse("http://example.com/")
	header := make(http.Header)
	header.Add("Foo", "bar")

	req := &http.Request{
		Method:     "GET",
		URL:        u,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     header,
		Host:       u.Host,
		RemoteAddr: "127.0.0.1:8000",
	}
	return req
}

func newBaseHttp() *Http {
	h := &Http{
		Method:  "GET",
		Cookies: "",
		Query:   "",
		URL:     "http://example.com/",
		Headers: map[string]string{"Foo": "bar"},
		Env:     map[string]string{"REMOTE_ADDR": "127.0.0.1", "REMOTE_PORT": "8000"},
	}
	return h
}

func NewRequest() testcase {
	return testcase{newBaseRequest(), newBaseHttp()}
}

func NewRequestIPV6() testcase {
	req := newBaseRequest()
	req.RemoteAddr = "[:1]:8000"

	h := newBaseHttp()
	h.Env = map[string]string{"REMOTE_ADDR": ":1", "REMOTE_PORT": "8000"}
	return testcase{req, h}
}

func NewRequestMultipleHeaders() testcase {
	req := newBaseRequest()
	req.Header.Add("Foo", "baz")

	h := newBaseHttp()
	h.Headers["Foo"] = "bar,baz"
	return testcase{req, h}
}

func NewSecureRequest() testcase {
	req := newBaseRequest()
	req.Header.Add("X-Forwarded-Proto", "https")

	h := newBaseHttp()
	h.URL = "https://example.com/"
	h.Headers["X-Forwarded-Proto"] = "https"
	return testcase{req, h}
}

func NewCookiesRequest() testcase {
	val := "foo=bar; bar=baz"
	req := newBaseRequest()
	req.Header.Add("Cookie", val)

	h := newBaseHttp()
	h.Cookies = val
	h.Headers["Cookie"] = val
	return testcase{req, h}
}

var newHttpTests = []testcase{
	NewRequest(),
	NewRequestIPV6(),
	NewRequestMultipleHeaders(),
	NewSecureRequest(),
	NewCookiesRequest(),
}

func TestNewHttp(t *testing.T) {
	for _, test := range newHttpTests {
		actual := NewHttp(test.request)
		if actual.Method != test.Method {
			t.Errorf("incorrect Method: got %s, want %s", actual.Method, test.Method)
		}
		if actual.Cookies != test.Cookies {
			t.Errorf("incorrect Cookies: got %s, want %s", actual.Cookies, test.Cookies)
		}
		if actual.Query != test.Query {
			t.Errorf("incorrect Query: got %s, want %s", actual.Query, test.Query)
		}
		if actual.URL != test.URL {
			t.Errorf("incorrect URL: got %s, want %s", actual.URL, test.URL)
		}
		if !reflect.DeepEqual(actual.Headers, test.Headers) {
			t.Errorf("incorrect Headers: got %+v, want %+v", actual.Headers, test.Headers)
		}
		if !reflect.DeepEqual(actual.Env, test.Env) {
			t.Errorf("incorrect Env: got %+v, want %+v", actual.Env, test.Env)
		}
		if !reflect.DeepEqual(actual.Data, test.Data) {
			t.Errorf("incorrect Data: got %+v, want %+v", actual.Data, test.Data)
		}
	}
}

var sanitizeQueryTests = []struct {
	input, output string
}{
	{"foo=bar", "foo=bar"},
	{"password=foo", "password=********"},
	{"passphrase=foo", "passphrase=********"},
	{"passwd=foo", "passwd=********"},
	{"secret=foo", "secret=********"},
	{"secretstuff=foo", "secretstuff=********"},
	{"foo=bar&secret=foo", "foo=bar&secret=********"},
	{"secret=foo&secret=bar", "secret=********"},
}

func parseQuery(q string) url.Values {
	r, _ := url.ParseQuery(q)
	return r
}

func TestSanitizeQuery(t *testing.T) {
	for _, test := range sanitizeQueryTests {
		actual := sanitizeQuery(parseQuery(test.input))
		expected := parseQuery(test.output)
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("incorrect sanitization: got %+v, want %+v", actual, expected)
		}
	}
}
