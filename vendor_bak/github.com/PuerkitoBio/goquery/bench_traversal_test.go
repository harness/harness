package goquery

import (
	"testing"
)

func BenchmarkFind(b *testing.B) {
	var n int

	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = DocB().Find("dd").Length()

		} else {
			DocB().Find("dd")
		}
	}
	b.Logf("Find=%d", n)
}

func BenchmarkFindWithinSelection(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("ul")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.Find("a[class]").Length()
		} else {
			sel.Find("a[class]")
		}
	}
	b.Logf("FindWithinSelection=%d", n)
}

func BenchmarkFindSelection(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("ul")
	sel2 := DocW().Find("span")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.FindSelection(sel2).Length()
		} else {
			sel.FindSelection(sel2)
		}
	}
	b.Logf("FindSelection=%d", n)
}

func BenchmarkFindNodes(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("ul")
	sel2 := DocW().Find("span")
	nodes := sel2.Nodes
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.FindNodes(nodes...).Length()
		} else {
			sel.FindNodes(nodes...)
		}
	}
	b.Logf("FindNodes=%d", n)
}

func BenchmarkContents(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find(".toclevel-1")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.Contents().Length()
		} else {
			sel.Contents()
		}
	}
	b.Logf("Contents=%d", n)
}

func BenchmarkContentsFiltered(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find(".toclevel-1")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.ContentsFiltered("a[href=\"#Examples\"]").Length()
		} else {
			sel.ContentsFiltered("a[href=\"#Examples\"]")
		}
	}
	b.Logf("ContentsFiltered=%d", n)
}

func BenchmarkChildren(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find(".toclevel-2")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.Children().Length()
		} else {
			sel.Children()
		}
	}
	b.Logf("Children=%d", n)
}

func BenchmarkChildrenFiltered(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("h3")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.ChildrenFiltered(".editsection").Length()
		} else {
			sel.ChildrenFiltered(".editsection")
		}
	}
	b.Logf("ChildrenFiltered=%d", n)
}

func BenchmarkParent(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("li")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.Parent().Length()
		} else {
			sel.Parent()
		}
	}
	b.Logf("Parent=%d", n)
}

func BenchmarkParentFiltered(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("li")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.ParentFiltered("ul[id]").Length()
		} else {
			sel.ParentFiltered("ul[id]")
		}
	}
	b.Logf("ParentFiltered=%d", n)
}

func BenchmarkParents(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("th a")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.Parents().Length()
		} else {
			sel.Parents()
		}
	}
	b.Logf("Parents=%d", n)
}

func BenchmarkParentsFiltered(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("th a")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.ParentsFiltered("tr").Length()
		} else {
			sel.ParentsFiltered("tr")
		}
	}
	b.Logf("ParentsFiltered=%d", n)
}

func BenchmarkParentsUntil(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("th a")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.ParentsUntil("table").Length()
		} else {
			sel.ParentsUntil("table")
		}
	}
	b.Logf("ParentsUntil=%d", n)
}

func BenchmarkParentsUntilSelection(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("th a")
	sel2 := DocW().Find("#content")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.ParentsUntilSelection(sel2).Length()
		} else {
			sel.ParentsUntilSelection(sel2)
		}
	}
	b.Logf("ParentsUntilSelection=%d", n)
}

func BenchmarkParentsUntilNodes(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("th a")
	sel2 := DocW().Find("#content")
	nodes := sel2.Nodes
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.ParentsUntilNodes(nodes...).Length()
		} else {
			sel.ParentsUntilNodes(nodes...)
		}
	}
	b.Logf("ParentsUntilNodes=%d", n)
}

func BenchmarkParentsFilteredUntil(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find(".toclevel-1 a")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.ParentsFilteredUntil(":nth-child(1)", "ul").Length()
		} else {
			sel.ParentsFilteredUntil(":nth-child(1)", "ul")
		}
	}
	b.Logf("ParentsFilteredUntil=%d", n)
}

func BenchmarkParentsFilteredUntilSelection(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find(".toclevel-1 a")
	sel2 := DocW().Find("ul")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.ParentsFilteredUntilSelection(":nth-child(1)", sel2).Length()
		} else {
			sel.ParentsFilteredUntilSelection(":nth-child(1)", sel2)
		}
	}
	b.Logf("ParentsFilteredUntilSelection=%d", n)
}

func BenchmarkParentsFilteredUntilNodes(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find(".toclevel-1 a")
	sel2 := DocW().Find("ul")
	nodes := sel2.Nodes
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.ParentsFilteredUntilNodes(":nth-child(1)", nodes...).Length()
		} else {
			sel.ParentsFilteredUntilNodes(":nth-child(1)", nodes...)
		}
	}
	b.Logf("ParentsFilteredUntilNodes=%d", n)
}

func BenchmarkSiblings(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("ul li:nth-child(1)")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.Siblings().Length()
		} else {
			sel.Siblings()
		}
	}
	b.Logf("Siblings=%d", n)
}

func BenchmarkSiblingsFiltered(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("ul li:nth-child(1)")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.SiblingsFiltered("[class]").Length()
		} else {
			sel.SiblingsFiltered("[class]")
		}
	}
	b.Logf("SiblingsFiltered=%d", n)
}

func BenchmarkNext(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("li:nth-child(1)")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.Next().Length()
		} else {
			sel.Next()
		}
	}
	b.Logf("Next=%d", n)
}

func BenchmarkNextFiltered(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("li:nth-child(1)")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.NextFiltered("[class]").Length()
		} else {
			sel.NextFiltered("[class]")
		}
	}
	b.Logf("NextFiltered=%d", n)
}

func BenchmarkNextAll(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("li:nth-child(3)")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.NextAll().Length()
		} else {
			sel.NextAll()
		}
	}
	b.Logf("NextAll=%d", n)
}

func BenchmarkNextAllFiltered(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("li:nth-child(3)")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.NextAllFiltered("[class]").Length()
		} else {
			sel.NextAllFiltered("[class]")
		}
	}
	b.Logf("NextAllFiltered=%d", n)
}

func BenchmarkPrev(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("li:last-child")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.Prev().Length()
		} else {
			sel.Prev()
		}
	}
	b.Logf("Prev=%d", n)
}

func BenchmarkPrevFiltered(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("li:last-child")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.PrevFiltered("[class]").Length()
		} else {
			sel.PrevFiltered("[class]")
		}
	}
	// There is one more Prev li with a class, compared to Next li with a class
	// (confirmed by looking at the HTML, this is ok)
	b.Logf("PrevFiltered=%d", n)
}

func BenchmarkPrevAll(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("li:nth-child(4)")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.PrevAll().Length()
		} else {
			sel.PrevAll()
		}
	}
	b.Logf("PrevAll=%d", n)
}

func BenchmarkPrevAllFiltered(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("li:nth-child(4)")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.PrevAllFiltered("[class]").Length()
		} else {
			sel.PrevAllFiltered("[class]")
		}
	}
	b.Logf("PrevAllFiltered=%d", n)
}

func BenchmarkNextUntil(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("li:first-child")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.NextUntil(":nth-child(4)").Length()
		} else {
			sel.NextUntil(":nth-child(4)")
		}
	}
	b.Logf("NextUntil=%d", n)
}

func BenchmarkNextUntilSelection(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("h2")
	sel2 := DocW().Find("ul")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.NextUntilSelection(sel2).Length()
		} else {
			sel.NextUntilSelection(sel2)
		}
	}
	b.Logf("NextUntilSelection=%d", n)
}

func BenchmarkNextUntilNodes(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("h2")
	sel2 := DocW().Find("p")
	nodes := sel2.Nodes
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.NextUntilNodes(nodes...).Length()
		} else {
			sel.NextUntilNodes(nodes...)
		}
	}
	b.Logf("NextUntilNodes=%d", n)
}

func BenchmarkPrevUntil(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("li:last-child")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.PrevUntil(":nth-child(4)").Length()
		} else {
			sel.PrevUntil(":nth-child(4)")
		}
	}
	b.Logf("PrevUntil=%d", n)
}

func BenchmarkPrevUntilSelection(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("h2")
	sel2 := DocW().Find("ul")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.PrevUntilSelection(sel2).Length()
		} else {
			sel.PrevUntilSelection(sel2)
		}
	}
	b.Logf("PrevUntilSelection=%d", n)
}

func BenchmarkPrevUntilNodes(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("h2")
	sel2 := DocW().Find("p")
	nodes := sel2.Nodes
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.PrevUntilNodes(nodes...).Length()
		} else {
			sel.PrevUntilNodes(nodes...)
		}
	}
	b.Logf("PrevUntilNodes=%d", n)
}

func BenchmarkNextFilteredUntil(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("h2")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.NextFilteredUntil("p", "div").Length()
		} else {
			sel.NextFilteredUntil("p", "div")
		}
	}
	b.Logf("NextFilteredUntil=%d", n)
}

func BenchmarkNextFilteredUntilSelection(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("h2")
	sel2 := DocW().Find("div")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.NextFilteredUntilSelection("p", sel2).Length()
		} else {
			sel.NextFilteredUntilSelection("p", sel2)
		}
	}
	b.Logf("NextFilteredUntilSelection=%d", n)
}

func BenchmarkNextFilteredUntilNodes(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("h2")
	sel2 := DocW().Find("div")
	nodes := sel2.Nodes
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.NextFilteredUntilNodes("p", nodes...).Length()
		} else {
			sel.NextFilteredUntilNodes("p", nodes...)
		}
	}
	b.Logf("NextFilteredUntilNodes=%d", n)
}

func BenchmarkPrevFilteredUntil(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("h2")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.PrevFilteredUntil("p", "div").Length()
		} else {
			sel.PrevFilteredUntil("p", "div")
		}
	}
	b.Logf("PrevFilteredUntil=%d", n)
}

func BenchmarkPrevFilteredUntilSelection(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("h2")
	sel2 := DocW().Find("div")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.PrevFilteredUntilSelection("p", sel2).Length()
		} else {
			sel.PrevFilteredUntilSelection("p", sel2)
		}
	}
	b.Logf("PrevFilteredUntilSelection=%d", n)
}

func BenchmarkPrevFilteredUntilNodes(b *testing.B) {
	var n int

	b.StopTimer()
	sel := DocW().Find("h2")
	sel2 := DocW().Find("div")
	nodes := sel2.Nodes
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.PrevFilteredUntilNodes("p", nodes...).Length()
		} else {
			sel.PrevFilteredUntilNodes("p", nodes...)
		}
	}
	b.Logf("PrevFilteredUntilNodes=%d", n)
}

func BenchmarkClosest(b *testing.B) {
	var n int

	b.StopTimer()
	sel := Doc().Find(".container-fluid")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.Closest(".pvk-content").Length()
		} else {
			sel.Closest(".pvk-content")
		}
	}
	b.Logf("Closest=%d", n)
}

func BenchmarkClosestSelection(b *testing.B) {
	var n int

	b.StopTimer()
	sel := Doc().Find(".container-fluid")
	sel2 := Doc().Find(".pvk-content")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.ClosestSelection(sel2).Length()
		} else {
			sel.ClosestSelection(sel2)
		}
	}
	b.Logf("ClosestSelection=%d", n)
}

func BenchmarkClosestNodes(b *testing.B) {
	var n int

	b.StopTimer()
	sel := Doc().Find(".container-fluid")
	nodes := Doc().Find(".pvk-content").Nodes
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if n == 0 {
			n = sel.ClosestNodes(nodes...).Length()
		} else {
			sel.ClosestNodes(nodes...)
		}
	}
	b.Logf("ClosestNodes=%d", n)
}
