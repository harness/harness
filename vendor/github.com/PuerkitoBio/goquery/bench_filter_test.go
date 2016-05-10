package goquery

import (
	"testing"
)

func BenchmarkFilter(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("li")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.Filter(".toclevel-1").Length()
		} else {
			sel.Filter(".toclevel-1")
		}
	}
	b.Logf("Filter=%d", n)
}

func BenchmarkNot(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("li")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.Not(".toclevel-2").Length()
		} else {
			sel.Filter(".toclevel-2")
		}
	}
	b.Logf("Not=%d", n)
}

func BenchmarkFilterFunction(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("li")
	f := func(i int, s *Selection) bool {
		return len(s.Get(0).Attr) > 0
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.FilterFunction(f).Length()
		} else {
			sel.FilterFunction(f)
		}
	}
	b.Logf("FilterFunction=%d", n)
}

func BenchmarkNotFunction(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("li")
	f := func(i int, s *Selection) bool {
		return len(s.Get(0).Attr) > 0
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.NotFunction(f).Length()
		} else {
			sel.NotFunction(f)
		}
	}
	b.Logf("NotFunction=%d", n)
}

func BenchmarkFilterNodes(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("li")
	sel2 := DocW().Find(".toclevel-2")
	nodes := sel2.Nodes
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.FilterNodes(nodes...).Length()
		} else {
			sel.FilterNodes(nodes...)
		}
	}
	b.Logf("FilterNodes=%d", n)
}

func BenchmarkNotNodes(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("li")
	sel2 := DocW().Find(".toclevel-1")
	nodes := sel2.Nodes
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.NotNodes(nodes...).Length()
		} else {
			sel.NotNodes(nodes...)
		}
	}
	b.Logf("NotNodes=%d", n)
}

func BenchmarkFilterSelection(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("li")
	sel2 := DocW().Find(".toclevel-2")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.FilterSelection(sel2).Length()
		} else {
			sel.FilterSelection(sel2)
		}
	}
	b.Logf("FilterSelection=%d", n)
}

func BenchmarkNotSelection(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("li")
	sel2 := DocW().Find(".toclevel-1")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.NotSelection(sel2).Length()
		} else {
			sel.NotSelection(sel2)
		}
	}
	b.Logf("NotSelection=%d", n)
}

func BenchmarkHas(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("h2")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.Has(".editsection").Length()
		} else {
			sel.Has(".editsection")
		}
	}
	b.Logf("Has=%d", n)
}

func BenchmarkHasNodes(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("li")
	sel2 := DocW().Find(".tocnumber")
	nodes := sel2.Nodes
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.HasNodes(nodes...).Length()
		} else {
			sel.HasNodes(nodes...)
		}
	}
	b.Logf("HasNodes=%d", n)
}

func BenchmarkHasSelection(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("li")
	sel2 := DocW().Find(".tocnumber")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.HasSelection(sel2).Length()
		} else {
			sel.HasSelection(sel2)
		}
	}
	b.Logf("HasSelection=%d", n)
}

func BenchmarkEnd(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("li").Has(".tocnumber")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.End().Length()
		} else {
			sel.End()
		}
	}
	b.Logf("End=%d", n)
}
