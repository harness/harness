package main

import (
	"encoding/hex"
	"fmt"
	"github.com/GeertJohan/go.rice"
	"github.com/davecgh/go-spew/spew"
	"log"
	"net/http"
	"os"
	"text/template"
)

func main() {
	box, err := rice.FindBox("example-files")
	if err != nil {
		log.Fatalf("error opening rice.Box: %s\n", err)
	}
	// spew.Dump(box)

	contentString, err := box.String("file.txt")
	if err != nil {
		log.Fatalf("could not read file contents as string: %s\n", err)
	}
	log.Printf("Read some file contents as string:\n%s\n", contentString)

	contentBytes, err := box.Bytes("file.txt")
	if err != nil {
		log.Fatalf("could not read file contents as byteSlice: %s\n", err)
	}
	log.Printf("Read some file contents as byteSlice:\n%s\n", hex.Dump(contentBytes))

	file, err := box.Open("file.txt")
	if err != nil {
		log.Fatalf("could not open file: %s\n", err)
	}
	spew.Dump(file)
	// debianFile, err := box.Open("debian-7.3.0-amd64-i386-netinst.iso")
	// if err != nil {
	// 	log.Fatalf("error opening file debian-7.3.0-amd64-i386-netinst.iso: %v", err)
	// }
	// info, err := debianFile.Stat()
	// if err != nil {
	// 	log.Fatalf("error doing stat for debian file: %v", err)
	// }
	// log.Printf("debian file was last modified at %v\n", info.ModTime())
	// log.Printf("debian file is %d bytes large\n", info.Size())

	// find/create a rice.Box
	templateBox, err := rice.FindBox("example-templates")
	if err != nil {
		log.Fatal(err)
	}
	// get file contents as string
	templateString, err := templateBox.String("message.tmpl")
	if err != nil {
		log.Fatal(err)
	}
	// parse and execute the template
	tmplMessage, err := template.New("message").Parse(templateString)
	if err != nil {
		log.Fatal(err)
	}
	tmplMessage.Execute(os.Stdout, map[string]string{"Message": "Hello, world!"})

	http.Handle("/", http.FileServer(box.HTTPBox()))
	go http.ListenAndServe(":8080", nil)
	fmt.Printf("Serving files on :8080, press ctrl-C to exit")
	select {}
}
