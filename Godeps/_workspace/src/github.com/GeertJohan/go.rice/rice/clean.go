package main

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strings"
)

func operationClean(pkg *build.Package) {
	filepath.Walk(pkg.Dir, func(filename string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("error walking pkg dir to clean files: %v\n", err)
			os.Exit(1)
		}
		if info.IsDir() {
			return nil
		}
		verbosef("checking file '%s'\n", filename)
		if strings.HasSuffix(filename, ".rice-box.go") ||
			strings.HasSuffix(filename, ".rice-box.syso") {
			err := os.Remove(filename)
			if err != nil {
				fmt.Printf("error removing file (%s): %s\n", filename, err)
				os.Exit(-1)
			}
			verbosef("removed file '%s'\n", filename)
		}
		return nil
	})
}
