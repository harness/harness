package main

import (
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

func findBoxes(pkg *build.Package) map[string]bool {
	// create one list of files for this package
	filenames := make([]string, 0, len(pkg.GoFiles)+len(pkg.CgoFiles))
	filenames = append(filenames, pkg.GoFiles...)
	filenames = append(filenames, pkg.CgoFiles...)

	// prepare regex to find calls to rice.FindBox(..)
	regexpBox, err := regexp.Compile(`rice\.(?:Must)?FindBox\(["` + "`" + `]{1}([a-zA-Z0-9\\/\.\-_]+)["` + "`" + `]{1}\)`)
	if err != nil {
		fmt.Printf("error compiling rice.FindBox regexp: %s\n", err)
		os.Exit(1)
	}

	// create map of boxes to embed
	var boxMap = make(map[string]bool)

	// loop over files, search for rice.FindBox(..) calls
	for _, filename := range filenames {
		// find full filepath
		fullpath := filepath.Join(pkg.Dir, filename)
		verbosef("scanning file %s\n", fullpath)

		// open source file
		file, err := os.Open(fullpath)
		if err != nil {
			fmt.Printf("error opening file '%s': %s\n", filename, err)
			os.Exit(1)
		}
		defer file.Close()

		// slurp source code
		fileData, err := ioutil.ReadAll(file)
		if err != nil {
			fmt.Printf("error reading file '%s': %s\n", filename, err)
			os.Exit(1)
		}

		// find rice.FindBox(..) calls
		matches := regexpBox.FindAllStringSubmatch(string(fileData), -1)
		for _, match := range matches {
			boxMap[match[1]] = true
			verbosef("\tfound box '%s'\n", match[1])
		}
	}

	return boxMap
}
