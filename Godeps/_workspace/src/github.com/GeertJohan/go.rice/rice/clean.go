package main

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strings"
)

func operationClean(pkg *build.Package) {
	for _, filename := range pkg.GoFiles {
		verbosef("checking file '%s'\n", filename)
		if strings.HasSuffix(filename, ".rice-box.go") || strings.HasSuffix(filename, ".rice-single.go") {
			err := os.Remove(filepath.Join(pkg.Dir, filename))
			if err != nil {
				fmt.Printf("error removing file (%s): %s\n", filename, err)
				os.Exit(-1)
			}
			verbosef("removed file '%s'\n", filename)
		}
	}
}
