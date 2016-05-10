package goquery

import (
	"testing"
)

func BenchmarkAdd(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocB().Find("dd")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.Add("h2[title]").Length()
		} else {
			sel.Add("h2[title]")
		}
	}
	b.Logf("Add=%d", n)
}

func BenchmarkAddSelection(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocB().Find("dd")
	sel2 := DocB().Find("h2[title]")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.AddSelection(sel2).Length()
		} else {
			sel.AddSelection(sel2)
		}
	}
	b.Logf("AddSelection=%d", n)
}

func BenchmarkAddNodes(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocB().Find("dd")
	sel2 := DocB().Find("h2[title]")
	nodes := sel2.Nodes
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.AddNodes(nodes...).Length()
		} else {
			sel.AddNodes(nodes...)
		}
	}
	b.Logf("AddNodes=%d", n)
}

func BenchmarkAndSelf(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocB().Find("dd").Parent()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.AndSelf().Length()
		} else {
			sel.AndSelf()
		}
	}
	b.Logf("AndSelf=%d", n)
}
