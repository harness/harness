package goquery

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"
)

func BenchmarkMetalReviewExample(b *testing.B) {
	var n int
	var buf bytes.Buffer

	b.StopTimer()
	doc := loadDoc("metalreview.html")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		doc.Find(".slider-row:nth-child(1) .slider-item").Each(func(i int, s *Selection) {
			var band, title string
			var score float64
			var e error

			n++
			// For each item found, get the band, title and score, and print it
			band = s.Find("strong").Text()
			title = s.Find("em").Text()
			if score, e = strconv.ParseFloat(s.Find(".score").Text(), 64); e != nil {
				// Not a valid float, ignore score
				if n <= 4 {
					buf.WriteString(fmt.Sprintf("Review %d: %s - %s.\n", i, band, title))
				}
			} else {
				// Print all, including score
				if n <= 4 {
					buf.WriteString(fmt.Sprintf("Review %d: %s - %s (%2.1f).\n", i, band, title, score))
				}
			}
		})
	}
	b.Log(buf.String())
	b.Logf("MetalReviewExample=%d", n)
}
