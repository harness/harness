package goquery

import (
	"testing"
)

func BenchmarkEach(b *testing.B) {
	var tmp, n int

	b.StopTimer()
	sel := DocW().Find("td")
	f := func(i int, s *Selection) {
		tmp++
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		sel.Each(f)
		if n == 0 {
			n = tmp
		}
	}
	b.Logf("Each=%d", n)
}

func BenchmarkMap(b *testing.B) {
	var tmp, n int

	b.StopTimer()
	sel := DocW().Find("td")
	f := func(i int, s *Selection) string {
		tmp++
		return string(tmp)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		sel.Map(f)
		if n == 0 {
			n = tmp
		}
	}
	b.Logf("Map=%d", n)
}

func BenchmarkEachWithBreak(b *testing.B) {
	var tmp, n int

	b.StopTimer()
	sel := DocW().Find("td")
	f := func(i int, s *Selection) bool {
		tmp++
		return tmp < 10
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tmp = 0
		sel.EachWithBreak(f)
		if n == 0 {
			n = tmp
		}
	}
	b.Logf("Each=%d", n)
}
