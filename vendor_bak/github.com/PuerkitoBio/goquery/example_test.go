package goquery_test

import (
	"fmt"
	"log"

	"github.com/PuerkitoBio/goquery"
)

// This example scrapes the reviews shown on the home page of metalsucks.net.
func Example() {
	// Load the HTML document
	doc, err := goquery.NewDocument("http://metalsucks.net")
	if err != nil {
		log.Fatal(err)
	}

	// Find the review items
	doc.Find(".reviews-wrap article .review-rhs").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the band and title
		band := s.Find("h3").Text()
		title := s.Find("i").Text()
		fmt.Printf("Review %d: %s - %s\n", i, band, title)
	})
	// To see the output of the Example while running the test suite (go test), simply
	// remove the leading "x" before Output on the next line. This will cause the
	// example to fail (all the "real" tests should pass).

	// xOutput: voluntarily fail the Example output.
}
