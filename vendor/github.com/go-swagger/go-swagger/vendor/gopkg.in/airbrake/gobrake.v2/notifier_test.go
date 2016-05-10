package gobrake_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"gopkg.in/airbrake/gobrake.v2"
)

func TestGobrake(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "gobrake")
}

var _ = Describe("Notifier", func() {
	var notifier *gobrake.Notifier
	var sentNotice *gobrake.Notice

	notify := func(e interface{}, req *http.Request) {
		notifier.Notify(e, req)
		notifier.Flush()
	}

	BeforeEach(func() {
		handler := func(w http.ResponseWriter, req *http.Request) {
			b, err := ioutil.ReadAll(req.Body)
			if err != nil {
				panic(err)
			}

			sentNotice = &gobrake.Notice{}
			err = json.Unmarshal(b, sentNotice)
			Expect(err).To(BeNil())

			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id":"123"}`))
		}
		server := httptest.NewServer(http.HandlerFunc(handler))

		notifier = gobrake.NewNotifier(1, "key")
		notifier.SetHost(server.URL)
	})

	It("reports error and backtrace", func() {
		notify("hello", nil)

		e := sentNotice.Errors[0]
		Expect(e.Type).To(Equal("string"))
		Expect(e.Message).To(Equal("hello"))
		Expect(e.Backtrace[0].File).To(ContainSubstring("notifier_test.go"))
	})

	It("Notice returns proper backtrace", func() {
		notice := notifier.Notice("hello", nil, 0)

		e := notice.Errors[0]
		Expect(e.Backtrace[0].File).To(ContainSubstring("notifier_test.go"))
	})

	It("reports context, env, session and params", func() {
		wanted := notifier.Notice("hello", nil, 3)
		wanted.Context["context1"] = "context1"
		wanted.Env["env1"] = "value1"
		wanted.Session["session1"] = "value1"
		wanted.Params["param1"] = "value1"

		id, err := notifier.SendNotice(wanted)
		Expect(err).To(BeNil())
		Expect(id).To(Equal("123"))

		Expect(sentNotice).To(Equal(wanted))
	})

	It("reports context using SetContext", func() {
		notifier.AddFilter(func(notice *gobrake.Notice) *gobrake.Notice {
			notice.Context["environment"] = "production"
			return notice
		})
		notify("hello", nil)

		Expect(sentNotice.Context["environment"]).To(Equal("production"))
	})

	It("reports request", func() {
		u, err := url.Parse("http://foo/bar")
		Expect(err).To(BeNil())

		req := &http.Request{
			URL: u,
			Header: http.Header{
				"h1":         {"h1v1", "h1v2"},
				"h2":         {"h2v1"},
				"User-Agent": {"my_user_agent"},
			},
			Form: url.Values{
				"f1": {"f1v1"},
				"f2": {"f2v1", "f2v2"},
			},
		}

		notify("hello", req)

		ctx := sentNotice.Context
		Expect(ctx["url"]).To(Equal("http://foo/bar"))
		Expect(ctx["userAgent"]).To(Equal("my_user_agent"))

		params := sentNotice.Params
		Expect(params["f1"]).To(Equal("f1v1"))
		Expect(params["f2"]).To(Equal([]interface{}{"f2v1", "f2v2"}))

		env := sentNotice.Env
		Expect(env["h1"]).To(Equal([]interface{}{"h1v1", "h1v2"}))
		Expect(env["h2"]).To(Equal("h2v1"))
	})

	It("collects and reports context", func() {
		notify("hello", nil)

		hostname, _ := os.Hostname()
		wd, _ := os.Getwd()
		Expect(sentNotice.Context["language"]).To(Equal(runtime.Version()))
		Expect(sentNotice.Context["os"]).To(Equal(runtime.GOOS))
		Expect(sentNotice.Context["architecture"]).To(Equal(runtime.GOARCH))
		Expect(sentNotice.Context["hostname"]).To(Equal(hostname))
		Expect(sentNotice.Context["rootDirectory"]).To(Equal(wd))
	})
})
