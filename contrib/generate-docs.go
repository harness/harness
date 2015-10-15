// +build ignore

// This program generates converts our markdown
// documentation to an html website.

package main

import (
	"bytes"
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/eknkc/amber"
	"github.com/russross/blackfriday"
)

var (
	input  = flag.String("input", "docs/README.md", "path to site index")
	output = flag.String("output", "_site/", "path to outpute website")
	html   = flag.String("template", "", "path to documentation template")
	name   = flag.String("name", "", "name of the site")
)

func main() {
	flag.Parse()

	// read the markdown file into a document element.
	document, err := toDocument(*input)
	if err != nil {
		log.Fatalf("Error opening %s. %s", *input, err)
	}

	// we assume this file is the sitemap, so we'll look
	// for the first ul element, and assume that contains
	// the site hierarchy.
	sitemap := document.Find("ul").First()

	site := Site{}
	site.Name = *name
	site.base = filepath.Dir(*input)

	// for each link in the sitemap we should attempt to
	// read the markdown file and add to our list of pages
	// to generate.
	sitemap.Find("li > a").EachWithBreak(func(i int, s *goquery.Selection) bool {
		page, err := toPage(&site, s)
		if err != nil {
			log.Fatalf("Error following link %s. %s", s.Text(), err)
		}
		if page != nil {
			site.Pages = append(site.Pages, page)
		}
		return true
	})
	site.Nav = &Nav{}
	site.Nav.elem = sitemap
	site.Nav.html, err = sitemap.Html()
	if err != nil {
		log.Fatal(err)
	}

	// compiles our template which is in amber/jade format
	templ := amber.MustCompileFile(*html, amber.DefaultOptions)

	// for each page in the sitemap we generate a
	// corresponding html file using the above template.
	for _, page := range site.Pages {
		path := filepath.Join(*output, page.Href)
		f, err := os.Create(path)
		if err != nil {
			log.Fatalf("Error creating file %s. %s", path, err)
		}
		defer f.Close()

		// correctly make the active page in the
		// navigation html snippet
		site.Nav.elem.Find("li > a").EachWithBreak(func(i int, s *goquery.Selection) bool {
			href, _ := s.Attr("href")
			if href == page.Href {
				s.Parent().AddClass("active")
			} else {
				s.Parent().RemoveClass("active")
			}
			return true
		})
		site.Nav.html, _ = site.Nav.elem.Html()

		data := map[string]interface{}{
			"Site": site,
			"Page": page,
		}
		err = templ.Execute(f, &data)
		if err != nil {
			log.Fatalf("Error generating template %s. %s", path, err)
		}
	}
}

type Site struct {
	Nav   *Nav
	Pages []*Page
	Name  string

	base string
}

type Nav struct {
	html string
	elem *goquery.Selection
}

func (n *Nav) HTML() template.HTML {
	return template.HTML(n.html)
}

type Page struct {
	Name string
	Href string
	html string
}

func (p *Page) HTML() template.HTML {
	return template.HTML(p.html)
}

// toDocument is a helper function that parses a
// markdown file, converts to html markup, and returns
// a document element.
func toDocument(filename string) (*goquery.Document, error) {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	raw = blackfriday.MarkdownCommon(raw)

	buf := bytes.NewBuffer(raw)
	return goquery.NewDocumentFromReader(buf)
}

// toPage is a helper function that accepts an anchor
// tag referencing a markdown file, parsing the markdown
// file and returning a page to be included in our docs.
func toPage(site *Site, el *goquery.Selection) (*Page, error) {

	// follow the link to see if this is a page
	// that should be added to our documentation.
	href, ok := el.Attr("href")
	if !ok || href == "#" {
		return nil, nil
	}

	// read the markdown file, convert to html and
	// read into a dom element.
	doc, err := toDocument(filepath.Join(site.base, href))
	if err != nil {
		return nil, err
	}

	// convert the extension from markdown to
	// html, in preparation for type conversion.
	href = strings.Replace(href, ".md", ".html", -1)
	el.SetAttr("href", href)

	page := &Page{}
	page.Href = href
	page.html, err = doc.Html()
	return page, err
}
