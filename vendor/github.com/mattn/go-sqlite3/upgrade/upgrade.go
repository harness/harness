// +build !cgo
// +build upgrade

package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func download(prefix string) (url string, content []byte, err error) {
	year := time.Now().Year()

	site := "https://www.sqlite.org/download.html"
	//fmt.Printf("scraping %v\n", site)
	doc, err := goquery.NewDocument(site)
	if err != nil {
		log.Fatal(err)
	}

	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		if strings.HasPrefix(s.Text(), prefix) {
			url = fmt.Sprintf("https://www.sqlite.org/%d/", year) + s.Text()
		}
	})

	if url == "" {
		return "", nil, fmt.Errorf("Unable to find prefix '%s' on sqlite.org", prefix)
	}

	fmt.Printf("Downloading %v\n", url)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	// Ready Body Content
	content, err = ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return "", nil, err
	}

	return url, content, nil
}

func mergeFile(src string, dst string) error {
	defer func() error {
		fmt.Printf("Removing: %s\n", src)
		err := os.Remove(src)

		if err != nil {
			return err
		}

		return nil
	}()

	// Open destination
	fdst, err := os.OpenFile(dst, os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer fdst.Close()

	// Read source content
	content, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}

	// Add Additional newline
	if _, err := fdst.WriteString("\n"); err != nil {
		return err
	}

	fmt.Printf("Merging: %s into %s\n", src, dst)
	if _, err = fdst.Write(content); err != nil {
		return err
	}

	return nil
}

func main() {
	fmt.Println("Go-SQLite3 Upgrade Tool")

	// Download Amalgamation
	_, amalgamation, err := download("sqlite-amalgamation-")
	if err != nil {
		fmt.Println("Failed to download: sqlite-amalgamation; %s", err)
	}

	// Download Source
	_, source, err := download("sqlite-src-")
	if err != nil {
		fmt.Println("Failed to download: sqlite-src; %s", err)
	}

	// Create Amalgamation Zip Reader
	rAmalgamation, err := zip.NewReader(bytes.NewReader(amalgamation), int64(len(amalgamation)))
	if err != nil {
		log.Fatal(err)
	}

	// Create Source Zip Reader
	rSource, err := zip.NewReader(bytes.NewReader(source), int64(len(source)))
	if err != nil {
		log.Fatal(err)
	}

	// Extract Amalgamation
	for _, zf := range rAmalgamation.File {
		var f *os.File
		switch path.Base(zf.Name) {
		case "sqlite3.c":
			f, err = os.Create("sqlite3-binding.c")
		case "sqlite3.h":
			f, err = os.Create("sqlite3-binding.h")
		case "sqlite3ext.h":
			f, err = os.Create("sqlite3ext.h")
		default:
			continue
		}
		if err != nil {
			log.Fatal(err)
		}
		zr, err := zf.Open()
		if err != nil {
			log.Fatal(err)
		}

		_, err = io.WriteString(f, "#ifndef USE_LIBSQLITE3\n")
		if err != nil {
			zr.Close()
			f.Close()
			log.Fatal(err)
		}
		scanner := bufio.NewScanner(zr)
		for scanner.Scan() {
			text := scanner.Text()
			if text == `#include "sqlite3.h"` {
				text = `#include "sqlite3-binding.h"`
			}
			_, err = fmt.Fprintln(f, text)
			if err != nil {
				break
			}
		}
		err = scanner.Err()
		if err != nil {
			zr.Close()
			f.Close()
			log.Fatal(err)
		}
		_, err = io.WriteString(f, "#else // USE_LIBSQLITE3\n // If users really want to link against the system sqlite3 we\n// need to make this file a noop.\n #endif")
		if err != nil {
			zr.Close()
			f.Close()
			log.Fatal(err)
		}
		zr.Close()
		f.Close()
		fmt.Printf("Extracted: %v\n", filepath.Base(f.Name()))
	}

	//Extract Source
	for _, zf := range rSource.File {
		var f *os.File
		switch path.Base(zf.Name) {
		case "userauth.c":
			f, err = os.Create("userauth.c")
		case "sqlite3userauth.h":
			f, err = os.Create("userauth.h")
		default:
			continue
		}
		if err != nil {
			log.Fatal(err)
		}
		zr, err := zf.Open()
		if err != nil {
			log.Fatal(err)
		}

		_, err = io.Copy(f, zr)
		if err != nil {
			log.Fatal(err)
		}

		zr.Close()
		f.Close()
		fmt.Printf("extracted %v\n", filepath.Base(f.Name()))
	}

	// Merge SQLite User Authentication into amalgamation
	if err := mergeFile("userauth.c", "sqlite3-binding.c"); err != nil {
		log.Fatal(err)
	}
	if err := mergeFile("userauth.h", "sqlite3-binding.h"); err != nil {
		log.Fatal(err)
	}

	os.Exit(0)
}
