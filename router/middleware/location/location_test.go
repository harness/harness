package location

import (
	"github.com/franela/goblin"
	"net/http"
	"reflect"
	"testing"
)

var mockHeader []string
var mockRequest *http.Request

var wronglyFormedHeader []string
var wronglyFormedRequest *http.Request

func init() {
	mockHeader = []string{"For= 110.0.2.2", "for = \"[::1]\"; Host=example.com; foR=10.2.3.4; pRoto =https ; By = 127.0.0.1"}
	mockRequest = &http.Request{Header: map[string][]string{"Forwarded": mockHeader}}
	wronglyFormedHeader = []string{"Fro= 110.0.2.2", "for = \"[:1]\"% Host=example:.com| foR=10.278.3.4% poto =https | Bi % 127.0.0.1", ""}
	wronglyFormedRequest = &http.Request{Header: map[string][]string{"Forwarded": wronglyFormedHeader}}
}

func TestParseForwardedHeadersProto(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("Parse proto Forwarded Headers", func() {
		g.It("Should parse a normal proto Forwarded header", func() {
			parsedHeader := parseHeader(mockRequest, "Forwarded", "proto")
			g.Assert("https" == parsedHeader[0]).IsTrue()
		})
		g.It("Should parse a normal for Forwarded header", func() {
			parsedHeader := parseHeader(mockRequest, "Forwarded", "for")
			g.Assert(reflect.DeepEqual([]string{"110.0.2.2", "\"[::1]\"", "10.2.3.4"}, parsedHeader)).IsTrue()
		})
		g.It("Should parse a normal host Forwarded header", func() {
			parsedHeader := parseHeader(mockRequest, "Forwarded", "host")
			g.Assert("example.com" == parsedHeader[0]).IsTrue()
		})
		g.It("Should parse a normal by Forwarded header", func() {
			parsedHeader := parseHeader(mockRequest, "Forwarded", "by")
			g.Assert("127.0.0.1" == parsedHeader[0]).IsTrue()
		})
		g.It("Should not crash if a wrongly formed Forwarder header is sent", func() {
			parsedHeader := parseHeader(wronglyFormedRequest, "Forwarded", "by")
			g.Assert(len(parsedHeader) == 0).IsTrue()
		})
	})
}
