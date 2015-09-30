package airbrake

import (
	"bytes"
	"errors"
	"net/http"
	"regexp"
	"testing"
	"time"
)

const API_KEY = ""

func TestError(t *testing.T) {
	Verbose = true
	ApiKey = API_KEY
	Endpoint = "https://api.airbrake.io/notifier_api/v2/notices"

	err := Error(errors.New("Test Error"), nil)
	if err != nil {

		t.Error(err)
	}

	time.Sleep(1e9)
}

func TestRequest(t *testing.T) {
	Verbose = true
	ApiKey = API_KEY
	Endpoint = "https://api.airbrake.io/notifier_api/v2/notices"

	request, _ := http.NewRequest("GET", "/some/path?a=1", bytes.NewBufferString(""))

	err := Error(errors.New("Test Error"), request)

	if err != nil {
		t.Error(err)
	}

	time.Sleep(1e9)
}

func TestNotify(t *testing.T) {
	Verbose = true
	ApiKey = API_KEY
	Endpoint = "https://api.airbrake.io/notifier_api/v2/notices"

	err := Notify(errors.New("Test Error"))

	if err != nil {
		t.Error(err)
	}

	time.Sleep(1e9)
}

// Make sure we match https://help.airbrake.io/kb/api-2/notifier-api-version-23
func TestTemplateV2(t *testing.T) {
	var p map[string]interface{}
	request, _ := http.NewRequest("GET", "/query?t=xxx&q=SHOW+x+BY+y+FROM+z&key=sesame&timezone=", nil)
	request.Header.Set("Host", "Zulu")
	request.Header.Set("Keep_Secret", "Sesame")
	PrettyParams = true
	defer func() { PrettyParams = false }()

	// Trigger and recover a panic, so that we have something to render.
	func() {
		defer func() {
			if r := recover(); r != nil {
				p = params(r.(error), request)
			}
		}()
		panic(errors.New("Boom!"))
	}()

	// Did that work?
	if p == nil {
		t.Fail()
	}

	// Crude backtrace check.
	if len(p["Backtrace"].([]Line)) < 3 {
		t.Fail()
	}

	// It's messy to generically test rendered backtrace, drop it.
	delete(p, "Backtrace")

	// Render the params.
	var b bytes.Buffer
	if err := tmpl.Execute(&b, p); err != nil {
		t.Fatalf("Template error: %s", err)
	}

	// Validate the <error> node.
	chunk := regexp.MustCompile(`(?s)<error>.*<backtrace>`).FindString(b.String())
	if chunk != `<error>
    <class>*errors.errorString</class>
    <message>Boom!</message>
    <backtrace>` {
		t.Fatal(chunk)
	}

	// Validate the <request> node.
	chunk = regexp.MustCompile(`(?s)<request>.*</request>`).FindString(b.String())
	if chunk != `<request>
    <url>/query?t=xxx&amp;q=SHOW+x+BY+y+FROM+z&amp;key=sesame&amp;timezone=</url>
    <component></component>
    <action></action>
    <params>
      <var key="q">SHOW x BY y FROM z</var>
      <var key="t">xxx</var>
    </params>
    <cgi-data>
      <var key="?q">SHOW x BY y FROM z</var>
      <var key="?t">xxx</var>
      <var key="Host">Zulu</var>
      <var key="Method">GET</var>
      <var key="Protocol">HTTP/1.1</var>
    </cgi-data>
  </request>` {
		t.Fatal(chunk)
	}
}
