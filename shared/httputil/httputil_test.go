package httputil

import (
	"github.com/franela/goblin"
	"net/http"
	"reflect"
	"testing"
)

func TestParseForwardedHeaders(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("Parse Forwarded Headers", func() {
		g.It("Should parse a normal Forwarded header", func() {
			modelHeader := forwardedHeader{For: []string{"110.0.2.2", "\"[::1]\"", "10.2.3.4"}, By: []string{"127.0.0.1"}, Proto: "https", Host: "example.com"}

			header := []string{"For= 110.0.2.2", "for = \"[::1]\"; Host=example.com; foR=10.2.3.4; pRoto =https ; By = 127.0.0.1"}
			parsedHeader := parseForwardedHeader(&http.Request{Header: map[string][]string{"Forwarded": header}})

			g.Assert(reflect.DeepEqual(parsedHeader, modelHeader)).IsTrue()

		})
	})

}
