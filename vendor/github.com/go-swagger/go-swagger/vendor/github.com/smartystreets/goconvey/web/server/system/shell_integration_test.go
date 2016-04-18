package system

import (
	"log"
	"path/filepath"
	"runtime"
	"strings"

	"testing"
)

func TestShellIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping potentially long-running integration test...")
		return
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)

	_, filename, _, _ := runtime.Caller(0)
	directory := filepath.Join(filepath.Dir(filename), "..", "watch", "integration_testing", "sub")
	packageName := "github.com/smartystreets/goconvey/web/server/watch/integration_testing/sub"

	shell := NewShell("go", "", true, "5s")
	output, err := shell.GoTest(directory, packageName, []string{}, []string{"-short"})

	if !strings.Contains(output, "PASS\n") || !strings.Contains(output, "ok") {
		t.Errorf("Expected output that resembed tests passing but got this instead: [%s]", output)
	}
	if err != nil {
		t.Error("Test run resulted in the following error:", err)
	}
}
