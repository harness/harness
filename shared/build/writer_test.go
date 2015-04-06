package build

import (
	"bytes"
	"testing"
)

func TestSetupDockerfile(t *testing.T) {
	var buf bytes.Buffer

	// wrap the buffer so we can analyze output
	w := writer{&buf, 0}

	w.WriteString("#DRONE:676f206275696c64\n")
	w.WriteString("#DRONE:676f2074657374202d76\n")
	w.WriteString("PASS\n")
	w.WriteString("ok  	github.com/garyburd/redigo/redis	0.113s\n")

	expected := `$ go build
$ go test -v
PASS
ok  	github.com/garyburd/redigo/redis	0.113s
`
	if expected != buf.String() {
		t.Errorf("Expected commands decoded and echoed correctly. got \n%s", buf.String())
	}
}
