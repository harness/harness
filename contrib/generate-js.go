// +build ignore

// This program minifies JavaScript files
// $ go run generate-js.go -dir scripts/ -out scripts/drone.min.js

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/dchest/jsmin"
)

var (
	dir = flag.String("dir", "scripts/", "")
	out = flag.String("o", "scripts/drone.min.js", "")
)

func main() {
	flag.Parse()

	var buf bytes.Buffer

	// walk the directory tree and write all
	// javascript files to the buffer.
	filepath.Walk(*dir, func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) != ".js" {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		// write the file name to the minified output
		fmt.Fprintf(&buf, "// %s\n", path)

		// copy the file to the buffer
		_, err = io.Copy(&buf, f)
		return err
	})

	// minifies the javascript
	data, err := jsmin.Minify(buf.Bytes())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// write the minified output
	ioutil.WriteFile(*out, data, 0700)
}
