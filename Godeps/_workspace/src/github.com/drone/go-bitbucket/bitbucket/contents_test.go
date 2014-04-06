package bitbucket

import (
	"fmt"
	"testing"
)

func Test_Contents(t *testing.T) {

	const testFile = "readme.rst"
	const testRev = "8f0fe25998516f460ce2a2a867b7298b3628dd23"

	// GET the latest revision for the repo

	// GET the README file for the repo & revision
	src, err := client.Sources.Find("atlassian", "jetbrains-bitbucket-connector", testRev, testFile)
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Println(src)
}
