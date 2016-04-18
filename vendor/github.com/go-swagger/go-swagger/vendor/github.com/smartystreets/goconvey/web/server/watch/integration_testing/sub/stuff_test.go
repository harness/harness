// This file's only purpose is to provide a realistic
// environment from which to run integration tests
// against the Watcher.
package sub

import (
	"fmt"
	"testing"
)

func TestStuff(t *testing.T) {
	if testing.Short() {
		return
	}

	fmt.Println()
}
