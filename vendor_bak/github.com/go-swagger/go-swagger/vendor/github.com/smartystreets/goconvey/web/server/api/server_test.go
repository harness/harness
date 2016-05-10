package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/smartystreets/goconvey/web/server/contract"
	"github.com/smartystreets/goconvey/web/server/messaging"
)

const initialRoot = "/root/gopath/src/github.com/smartystreets/project"
const nonexistentRoot = "I don't exist"
const unreadableContent = "!!error!!"

func TestHTTPServer(t *testing.T) {
	// TODO: fix the skipped tests...

	Convey("Subject: HttpServer responds to requests appropriately", t, func() {
		fixture := newServerFixture()

		Convey("Before any update is recieved", func() {
			Convey("When the update is requested", func() {
				update, _ := fixture.RequestLatest()

				Convey("No panic should occur", func() {
					So(func() { fixture.RequestLatest() }, ShouldNotPanic)
				})

				Convey("The update will be empty", func() {
					So(update, ShouldResemble, new(contract.CompleteOutput))
				})
			})
		})

		Convey("Given an update is received", func() {
			fixture.ReceiveUpdate("", &contract.CompleteOutput{Revision: "asdf"})

			Convey("When the update is requested", func() {
				update, response := fixture.RequestLatest()

				Convey("The server returns it", func() {
					So(update, ShouldResemble, &contract.CompleteOutput{Revision: "asdf"})
				})

				Convey("The server returns 200", func() {
					So(response.Code, ShouldEqual, http.StatusOK)
				})

				Convey("The server should include important cache-related headers", func() {
					So(len(response.HeaderMap), ShouldEqual, 4)
					So(response.HeaderMap["Content-Type"][0], ShouldEqual, "application/json")
					So(response.HeaderMap["Cache-Control"][0], ShouldEqual, "no-cache, no-store, must-revalidate")
					So(response.HeaderMap["Pragma"][0], ShouldEqual, "no-cache")
					So(response.HeaderMap["Expires"][0], ShouldEqual, "0")
				})
			})
		})

		Convey("When the root watch is queried", func() {
			root, status := fixture.QueryRootWatch(false)

			SkipConvey("The server returns it", func() {
				So(root, ShouldEqual, initialRoot)
			})

			Convey("The server returns HTTP 200 - OK", func() {
				So(status, ShouldEqual, http.StatusOK)
			})
		})

		SkipConvey("When the root watch is adjusted", func() {

			Convey("But the request has no root parameter", func() {
				status, body := fixture.AdjustRootWatchMalformed()

				Convey("The server returns HTTP 400 - Bad Input", func() {
					So(status, ShouldEqual, http.StatusBadRequest)
				})

				Convey("The body should contain a helpful error message", func() {
					So(body, ShouldEqual, "No 'root' query string parameter included!")
				})

				Convey("The server should not change the existing root", func() {
					root, _ := fixture.QueryRootWatch(false)
					So(root, ShouldEqual, initialRoot)
				})
			})

			Convey("But the root parameter is empty", func() {
				status, body := fixture.AdjustRootWatch("")

				Convey("The server returns HTTP 400 - Bad Input", func() {
					So(status, ShouldEqual, http.StatusBadRequest)
				})

				Convey("The server should provide a helpful error message", func() {
					So(body, ShouldEqual, "You must provide a non-blank path.")
				})

				Convey("The server should not change the existing root", func() {
					root, _ := fixture.QueryRootWatch(false)
					So(root, ShouldEqual, initialRoot)
				})
			})

			Convey("And the new root exists", func() {
				status, body := fixture.AdjustRootWatch(initialRoot + "/package")

				Convey("The server returns HTTP 200 - OK", func() {
					So(status, ShouldEqual, http.StatusOK)
				})

				Convey("The body should NOT contain any error message or content", func() {
					So(body, ShouldEqual, "")
				})

				Convey("The server informs the watcher of the new root", func() {
					root, _ := fixture.QueryRootWatch(false)
					So(root, ShouldEqual, initialRoot+"/package")
				})
			})

			Convey("And the new root does NOT exist", func() {
				status, body := fixture.AdjustRootWatch(nonexistentRoot)

				Convey("The server returns HTTP 404 - Not Found", func() {
					So(status, ShouldEqual, http.StatusNotFound)
				})

				Convey("The body should contain a helpful error message", func() {
					So(body, ShouldEqual, fmt.Sprintf("Directory does not exist: '%s'", nonexistentRoot))
				})

				Convey("The server should not change the existing root", func() {
					root, _ := fixture.QueryRootWatch(false)
					So(root, ShouldEqual, initialRoot)
				})
			})
		})

		SkipConvey("When a packge is ignored", func() {

			Convey("But the request has no path parameter", func() {
				status, body := fixture.IgnoreMalformed()

				Convey("The server returns HTTP 400 - Bad Input", func() {
					So(status, ShouldEqual, http.StatusBadRequest)
				})

				Convey("The body should contain a helpful error message", func() {
					So(body, ShouldEqual, "No 'paths' query string parameter included!")
				})

				SkipConvey("The server should not ignore anything", func() {
					// So(fixture.watcher.ignored, ShouldEqual, "")
				})
			})

			Convey("But the request is blank", func() {
				status, body := fixture.Ignore("")

				Convey("The server returns HTTP 400 - Bad Input", func() {
					So(status, ShouldEqual, http.StatusBadRequest)
				})

				Convey("The body should contain a helpful error message", func() {
					So(body, ShouldEqual, "You must provide a non-blank path.")
				})
			})

			Convey("And the request is well formed", func() {
				status, _ := fixture.Ignore(initialRoot)

				SkipConvey("The server informs the watcher", func() {
					// So(fixture.watcher.ignored, ShouldEqual, initialRoot)
				})
				Convey("The server returns HTTP 200 - OK", func() {
					So(status, ShouldEqual, http.StatusOK)
				})
			})
		})

		SkipConvey("When a package is reinstated", func() {
			Convey("But the request has no path parameter", func() {
				status, body := fixture.ReinstateMalformed()

				Convey("The server returns HTTP 400 - Bad Input", func() {
					So(status, ShouldEqual, http.StatusBadRequest)
				})

				Convey("The body should contain a helpful error message", func() {
					So(body, ShouldEqual, "No 'paths' query string parameter included!")
				})

				SkipConvey("The server should not ignore anything", func() {
					// So(fixture.watcher.reinstated, ShouldEqual, "")
				})
			})

			Convey("But the request is blank", func() {
				status, body := fixture.Reinstate("")

				Convey("The server returns HTTP 400 - Bad Input", func() {
					So(status, ShouldEqual, http.StatusBadRequest)
				})

				Convey("The body should contain a helpful error message", func() {
					So(body, ShouldEqual, "You must provide a non-blank path.")
				})
			})

			Convey("And the request is well formed", func() {
				status, _ := fixture.Reinstate(initialRoot)

				SkipConvey("The server informs the watcher", func() {
					// So(fixture.watcher.reinstated, ShouldEqual, initialRoot)
				})
				Convey("The server returns HTTP 200 - OK", func() {
					So(status, ShouldEqual, http.StatusOK)
				})
			})
		})

		Convey("When the status of the executor is requested", func() {
			fixture.executor.status = "blah blah blah"
			statusCode, statusBody := fixture.RequestExecutorStatus()

			Convey("The server asks the executor its status and returns it", func() {
				So(statusBody, ShouldEqual, "blah blah blah")
			})

			Convey("The server returns HTTP 200 - OK", func() {
				So(statusCode, ShouldEqual, http.StatusOK)
			})
		})

		Convey("When a manual execution of the test packages is requested", func() {
			status := fixture.ManualExecution()
			update, _ := fixture.RequestLatest()

			SkipConvey("The server invokes the executor using the watcher's listing and save the result", func() {
				So(update, ShouldResemble, &contract.CompleteOutput{Revision: initialRoot})
			})

			Convey("The server returns HTTP 200 - OK", func() {
				So(status, ShouldEqual, http.StatusOK)
			})
		})

		SkipConvey("When the pause setting is toggled via the server", func() {
			paused := fixture.TogglePause()

			SkipConvey("The pause channel buffer should have a true value", func() {
				// var value bool
				// select {
				// case value = <-fixture.pauseUpdate:
				// default:
				// }
				// So(value, ShouldBeTrue)
			})

			Convey("The latest results should show that the server is paused", func() {
				fixture.ReceiveUpdate("", &contract.CompleteOutput{Revision: "asdf"})
				update, _ := fixture.RequestLatest()

				So(update.Paused, ShouldBeTrue)
			})

			Convey("The toggle handler should return its new status", func() {
				So(paused, ShouldEqual, "true")
			})
		})
	})
}

/********* Server Fixture *********/

type ServerFixture struct {
	server       *HTTPServer
	watcher      chan messaging.WatcherCommand
	executor     *FakeExecutor
	statusUpdate chan bool
}

func (self *ServerFixture) ReceiveUpdate(root string, update *contract.CompleteOutput) {
	self.server.ReceiveUpdate(root, update)
}

func (self *ServerFixture) RequestLatest() (*contract.CompleteOutput, *httptest.ResponseRecorder) {
	request, _ := http.NewRequest("GET", "http://localhost:8080/results", nil)
	response := httptest.NewRecorder()

	self.server.Results(response, request)

	decoder := json.NewDecoder(strings.NewReader(response.Body.String()))
	update := new(contract.CompleteOutput)
	decoder.Decode(update)
	return update, response
}

func (self *ServerFixture) QueryRootWatch(newclient bool) (string, int) {
	url := "http://localhost:8080/watch"
	if newclient {
		url += "?newclient=1"
	}
	request, _ := http.NewRequest("GET", url, nil)
	response := httptest.NewRecorder()

	self.server.Watch(response, request)

	return strings.TrimSpace(response.Body.String()), response.Code
}

func (self *ServerFixture) AdjustRootWatchMalformed() (status int, body string) {
	request, _ := http.NewRequest("POST", "http://localhost:8080/watch", nil)
	response := httptest.NewRecorder()

	self.server.Watch(response, request)

	status, body = response.Code, strings.TrimSpace(response.Body.String())
	return
}

func (self *ServerFixture) AdjustRootWatch(newRoot string) (status int, body string) {
	escapedRoot := url.QueryEscape(newRoot)
	request, _ := http.NewRequest("POST", "http://localhost:8080/watch?root="+escapedRoot, nil)
	response := httptest.NewRecorder()

	self.server.Watch(response, request)

	status, body = response.Code, strings.TrimSpace(response.Body.String())
	return
}

func (self *ServerFixture) IgnoreMalformed() (status int, body string) {
	request, _ := http.NewRequest("POST", "http://localhost:8080/ignore", nil)
	response := httptest.NewRecorder()

	self.server.Ignore(response, request)

	status, body = response.Code, strings.TrimSpace(response.Body.String())
	return
}

func (self *ServerFixture) Ignore(folder string) (status int, body string) {
	escapedFolder := url.QueryEscape(folder)
	request, _ := http.NewRequest("POST", "http://localhost:8080/ignore?paths="+escapedFolder, nil)
	response := httptest.NewRecorder()

	self.server.Ignore(response, request)

	status, body = response.Code, strings.TrimSpace(response.Body.String())
	return
}

func (self *ServerFixture) ReinstateMalformed() (status int, body string) {
	request, _ := http.NewRequest("POST", "http://localhost:8080/reinstate", nil)
	response := httptest.NewRecorder()

	self.server.Reinstate(response, request)

	status, body = response.Code, strings.TrimSpace(response.Body.String())
	return
}

func (self *ServerFixture) Reinstate(folder string) (status int, body string) {
	escapedFolder := url.QueryEscape(folder)
	request, _ := http.NewRequest("POST", "http://localhost:8080/reinstate?paths="+escapedFolder, nil)
	response := httptest.NewRecorder()

	self.server.Reinstate(response, request)

	status, body = response.Code, strings.TrimSpace(response.Body.String())
	return
}

func (self *ServerFixture) SetExecutorStatus(status string) {
	// self.executor.status = status
	// select {
	// case self.executor.statusUpdate <- make(chan string):
	// default:
	// }
}

func (self *ServerFixture) RequestExecutorStatus() (code int, status string) {
	request, _ := http.NewRequest("GET", "http://localhost:8080/status", nil)
	response := httptest.NewRecorder()

	self.server.Status(response, request)

	code, status = response.Code, strings.TrimSpace(response.Body.String())
	return
}

func (self *ServerFixture) ManualExecution() int {
	request, _ := http.NewRequest("POST", "http://localhost:8080/execute", nil)
	response := httptest.NewRecorder()

	self.server.Execute(response, request)
	nap, _ := time.ParseDuration("100ms")
	time.Sleep(nap)
	return response.Code
}

func (self *ServerFixture) TogglePause() string {
	request, _ := http.NewRequest("POST", "http://localhost:8080/pause", nil)
	response := httptest.NewRecorder()

	self.server.TogglePause(response, request)

	return response.Body.String()
}

func newServerFixture() *ServerFixture {
	self := new(ServerFixture)
	self.watcher = make(chan messaging.WatcherCommand)
	// self.watcher.SetRootWatch(initialRoot)
	statusUpdate := make(chan chan string)
	self.executor = newFakeExecutor("", statusUpdate)
	self.server = NewHTTPServer("initial-working-dir", self.watcher, self.executor, statusUpdate)
	return self
}

/********* Fake Executor *********/

type FakeExecutor struct {
	status       string
	executed     bool
	statusFlag   bool
	statusUpdate chan chan string
}

func (self *FakeExecutor) Status() string {
	return self.status
}

func (self *FakeExecutor) ClearStatusFlag() bool {
	hasNewStatus := self.statusFlag
	self.statusFlag = false
	return hasNewStatus
}

func (self *FakeExecutor) ExecuteTests(watched []*contract.Package) *contract.CompleteOutput {
	output := new(contract.CompleteOutput)
	output.Revision = watched[0].Path
	return output
}

func newFakeExecutor(status string, statusUpdate chan chan string) *FakeExecutor {
	self := new(FakeExecutor)
	self.status = status
	self.statusUpdate = statusUpdate
	return self
}
