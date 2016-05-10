package goquery

import (
	"strings"
	"testing"
)

func TestFind(t *testing.T) {
	sel := Doc().Find("div.row-fluid")
	assertLength(t, sel.Nodes, 9)
}

func TestFindRollback(t *testing.T) {
	sel := Doc().Find("div.row-fluid")
	sel2 := sel.Find("a").End()
	assertEqual(t, sel, sel2)
}

func TestFindNotSelf(t *testing.T) {
	sel := Doc().Find("h1").Find("h1")
	assertLength(t, sel.Nodes, 0)
}

func TestFindInvalidSelector(t *testing.T) {
	defer assertPanic(t)
	Doc().Find(":+ ^")
}

func TestChainedFind(t *testing.T) {
	sel := Doc().Find("div.hero-unit").Find(".row-fluid")
	assertLength(t, sel.Nodes, 4)
}

func TestChildren(t *testing.T) {
	sel := Doc().Find(".pvk-content").Children()
	assertLength(t, sel.Nodes, 5)
}

func TestChildrenRollback(t *testing.T) {
	sel := Doc().Find(".pvk-content")
	sel2 := sel.Children().End()
	assertEqual(t, sel, sel2)
}

func TestContents(t *testing.T) {
	sel := Doc().Find(".pvk-content").Contents()
	assertLength(t, sel.Nodes, 13)
}

func TestContentsRollback(t *testing.T) {
	sel := Doc().Find(".pvk-content")
	sel2 := sel.Contents().End()
	assertEqual(t, sel, sel2)
}

func TestChildrenFiltered(t *testing.T) {
	sel := Doc().Find(".pvk-content").ChildrenFiltered(".hero-unit")
	assertLength(t, sel.Nodes, 1)
}

func TestChildrenFilteredRollback(t *testing.T) {
	sel := Doc().Find(".pvk-content")
	sel2 := sel.ChildrenFiltered(".hero-unit").End()
	assertEqual(t, sel, sel2)
}

func TestContentsFiltered(t *testing.T) {
	sel := Doc().Find(".pvk-content").ContentsFiltered(".hero-unit")
	assertLength(t, sel.Nodes, 1)
}

func TestContentsFilteredRollback(t *testing.T) {
	sel := Doc().Find(".pvk-content")
	sel2 := sel.ContentsFiltered(".hero-unit").End()
	assertEqual(t, sel, sel2)
}

func TestChildrenFilteredNone(t *testing.T) {
	sel := Doc().Find(".pvk-content").ChildrenFiltered("a.btn")
	assertLength(t, sel.Nodes, 0)
}

func TestParent(t *testing.T) {
	sel := Doc().Find(".container-fluid").Parent()
	assertLength(t, sel.Nodes, 3)
}

func TestParentRollback(t *testing.T) {
	sel := Doc().Find(".container-fluid")
	sel2 := sel.Parent().End()
	assertEqual(t, sel, sel2)
}

func TestParentBody(t *testing.T) {
	sel := Doc().Find("body").Parent()
	assertLength(t, sel.Nodes, 1)
}

func TestParentFiltered(t *testing.T) {
	sel := Doc().Find(".container-fluid").ParentFiltered(".hero-unit")
	assertLength(t, sel.Nodes, 1)
	assertClass(t, sel, "hero-unit")
}

func TestParentFilteredRollback(t *testing.T) {
	sel := Doc().Find(".container-fluid")
	sel2 := sel.ParentFiltered(".hero-unit").End()
	assertEqual(t, sel, sel2)
}

func TestParents(t *testing.T) {
	sel := Doc().Find(".container-fluid").Parents()
	assertLength(t, sel.Nodes, 8)
}

func TestParentsOrder(t *testing.T) {
	sel := Doc().Find("#cf2").Parents()
	assertLength(t, sel.Nodes, 6)
	assertSelectionIs(t, sel, ".hero-unit", ".pvk-content", "div.row-fluid", "#cf1", "body", "html")
}

func TestParentsRollback(t *testing.T) {
	sel := Doc().Find(".container-fluid")
	sel2 := sel.Parents().End()
	assertEqual(t, sel, sel2)
}

func TestParentsFiltered(t *testing.T) {
	sel := Doc().Find(".container-fluid").ParentsFiltered("body")
	assertLength(t, sel.Nodes, 1)
}

func TestParentsFilteredRollback(t *testing.T) {
	sel := Doc().Find(".container-fluid")
	sel2 := sel.ParentsFiltered("body").End()
	assertEqual(t, sel, sel2)
}

func TestParentsUntil(t *testing.T) {
	sel := Doc().Find(".container-fluid").ParentsUntil("body")
	assertLength(t, sel.Nodes, 6)
}

func TestParentsUntilRollback(t *testing.T) {
	sel := Doc().Find(".container-fluid")
	sel2 := sel.ParentsUntil("body").End()
	assertEqual(t, sel, sel2)
}

func TestParentsUntilSelection(t *testing.T) {
	sel := Doc().Find(".container-fluid")
	sel2 := Doc().Find(".pvk-content")
	sel = sel.ParentsUntilSelection(sel2)
	assertLength(t, sel.Nodes, 3)
}

func TestParentsUntilSelectionRollback(t *testing.T) {
	sel := Doc().Find(".container-fluid")
	sel2 := Doc().Find(".pvk-content")
	sel2 = sel.ParentsUntilSelection(sel2).End()
	assertEqual(t, sel, sel2)
}

func TestParentsUntilNodes(t *testing.T) {
	sel := Doc().Find(".container-fluid")
	sel2 := Doc().Find(".pvk-content, .hero-unit")
	sel = sel.ParentsUntilNodes(sel2.Nodes...)
	assertLength(t, sel.Nodes, 2)
}

func TestParentsUntilNodesRollback(t *testing.T) {
	sel := Doc().Find(".container-fluid")
	sel2 := Doc().Find(".pvk-content, .hero-unit")
	sel2 = sel.ParentsUntilNodes(sel2.Nodes...).End()
	assertEqual(t, sel, sel2)
}

func TestParentsFilteredUntil(t *testing.T) {
	sel := Doc().Find(".container-fluid").ParentsFilteredUntil(".pvk-content", "body")
	assertLength(t, sel.Nodes, 2)
}

func TestParentsFilteredUntilRollback(t *testing.T) {
	sel := Doc().Find(".container-fluid")
	sel2 := sel.ParentsFilteredUntil(".pvk-content", "body").End()
	assertEqual(t, sel, sel2)
}

func TestParentsFilteredUntilSelection(t *testing.T) {
	sel := Doc().Find(".container-fluid")
	sel2 := Doc().Find(".row-fluid")
	sel = sel.ParentsFilteredUntilSelection("div", sel2)
	assertLength(t, sel.Nodes, 3)
}

func TestParentsFilteredUntilSelectionRollback(t *testing.T) {
	sel := Doc().Find(".container-fluid")
	sel2 := Doc().Find(".row-fluid")
	sel2 = sel.ParentsFilteredUntilSelection("div", sel2).End()
	assertEqual(t, sel, sel2)
}

func TestParentsFilteredUntilNodes(t *testing.T) {
	sel := Doc().Find(".container-fluid")
	sel2 := Doc().Find(".row-fluid")
	sel = sel.ParentsFilteredUntilNodes("body", sel2.Nodes...)
	assertLength(t, sel.Nodes, 1)
}

func TestParentsFilteredUntilNodesRollback(t *testing.T) {
	sel := Doc().Find(".container-fluid")
	sel2 := Doc().Find(".row-fluid")
	sel2 = sel.ParentsFilteredUntilNodes("body", sel2.Nodes...).End()
	assertEqual(t, sel, sel2)
}

func TestSiblings(t *testing.T) {
	sel := Doc().Find("h1").Siblings()
	assertLength(t, sel.Nodes, 1)
}

func TestSiblingsRollback(t *testing.T) {
	sel := Doc().Find("h1")
	sel2 := sel.Siblings().End()
	assertEqual(t, sel, sel2)
}

func TestSiblings2(t *testing.T) {
	sel := Doc().Find(".pvk-gutter").Siblings()
	assertLength(t, sel.Nodes, 9)
}

func TestSiblings3(t *testing.T) {
	sel := Doc().Find("body>.container-fluid").Siblings()
	assertLength(t, sel.Nodes, 0)
}

func TestSiblingsFiltered(t *testing.T) {
	sel := Doc().Find(".pvk-gutter").SiblingsFiltered(".pvk-content")
	assertLength(t, sel.Nodes, 3)
}

func TestSiblingsFilteredRollback(t *testing.T) {
	sel := Doc().Find(".pvk-gutter")
	sel2 := sel.SiblingsFiltered(".pvk-content").End()
	assertEqual(t, sel, sel2)
}

func TestNext(t *testing.T) {
	sel := Doc().Find("h1").Next()
	assertLength(t, sel.Nodes, 1)
}

func TestNextRollback(t *testing.T) {
	sel := Doc().Find("h1")
	sel2 := sel.Next().End()
	assertEqual(t, sel, sel2)
}

func TestNext2(t *testing.T) {
	sel := Doc().Find(".close").Next()
	assertLength(t, sel.Nodes, 1)
}

func TestNextNone(t *testing.T) {
	sel := Doc().Find("small").Next()
	assertLength(t, sel.Nodes, 0)
}

func TestNextFiltered(t *testing.T) {
	sel := Doc().Find(".container-fluid").NextFiltered("div")
	assertLength(t, sel.Nodes, 2)
}

func TestNextFilteredRollback(t *testing.T) {
	sel := Doc().Find(".container-fluid")
	sel2 := sel.NextFiltered("div").End()
	assertEqual(t, sel, sel2)
}

func TestNextFiltered2(t *testing.T) {
	sel := Doc().Find(".container-fluid").NextFiltered("[ng-view]")
	assertLength(t, sel.Nodes, 1)
}

func TestPrev(t *testing.T) {
	sel := Doc().Find(".red").Prev()
	assertLength(t, sel.Nodes, 1)
	assertClass(t, sel, "green")
}

func TestPrevRollback(t *testing.T) {
	sel := Doc().Find(".red")
	sel2 := sel.Prev().End()
	assertEqual(t, sel, sel2)
}

func TestPrev2(t *testing.T) {
	sel := Doc().Find(".row-fluid").Prev()
	assertLength(t, sel.Nodes, 5)
}

func TestPrevNone(t *testing.T) {
	sel := Doc().Find("h2").Prev()
	assertLength(t, sel.Nodes, 0)
}

func TestPrevFiltered(t *testing.T) {
	sel := Doc().Find(".row-fluid").PrevFiltered(".row-fluid")
	assertLength(t, sel.Nodes, 5)
}

func TestPrevFilteredRollback(t *testing.T) {
	sel := Doc().Find(".row-fluid")
	sel2 := sel.PrevFiltered(".row-fluid").End()
	assertEqual(t, sel, sel2)
}

func TestNextAll(t *testing.T) {
	sel := Doc().Find("#cf2 div:nth-child(1)").NextAll()
	assertLength(t, sel.Nodes, 3)
}

func TestNextAllRollback(t *testing.T) {
	sel := Doc().Find("#cf2 div:nth-child(1)")
	sel2 := sel.NextAll().End()
	assertEqual(t, sel, sel2)
}

func TestNextAll2(t *testing.T) {
	sel := Doc().Find("div[ng-cloak]").NextAll()
	assertLength(t, sel.Nodes, 1)
}

func TestNextAllNone(t *testing.T) {
	sel := Doc().Find(".footer").NextAll()
	assertLength(t, sel.Nodes, 0)
}

func TestNextAllFiltered(t *testing.T) {
	sel := Doc().Find("#cf2 .row-fluid").NextAllFiltered("[ng-cloak]")
	assertLength(t, sel.Nodes, 2)
}

func TestNextAllFilteredRollback(t *testing.T) {
	sel := Doc().Find("#cf2 .row-fluid")
	sel2 := sel.NextAllFiltered("[ng-cloak]").End()
	assertEqual(t, sel, sel2)
}

func TestNextAllFiltered2(t *testing.T) {
	sel := Doc().Find(".close").NextAllFiltered("h4")
	assertLength(t, sel.Nodes, 1)
}

func TestPrevAll(t *testing.T) {
	sel := Doc().Find("[ng-view]").PrevAll()
	assertLength(t, sel.Nodes, 2)
}

func TestPrevAllOrder(t *testing.T) {
	sel := Doc().Find("[ng-view]").PrevAll()
	assertLength(t, sel.Nodes, 2)
	assertSelectionIs(t, sel, "#cf4", "#cf3")
}

func TestPrevAllRollback(t *testing.T) {
	sel := Doc().Find("[ng-view]")
	sel2 := sel.PrevAll().End()
	assertEqual(t, sel, sel2)
}

func TestPrevAll2(t *testing.T) {
	sel := Doc().Find(".pvk-gutter").PrevAll()
	assertLength(t, sel.Nodes, 6)
}

func TestPrevAllFiltered(t *testing.T) {
	sel := Doc().Find(".pvk-gutter").PrevAllFiltered(".pvk-content")
	assertLength(t, sel.Nodes, 3)
}

func TestPrevAllFilteredRollback(t *testing.T) {
	sel := Doc().Find(".pvk-gutter")
	sel2 := sel.PrevAllFiltered(".pvk-content").End()
	assertEqual(t, sel, sel2)
}

func TestNextUntil(t *testing.T) {
	sel := Doc().Find(".alert a").NextUntil("p")
	assertLength(t, sel.Nodes, 1)
	assertSelectionIs(t, sel, "h4")
}

func TestNextUntil2(t *testing.T) {
	sel := Doc().Find("#cf2-1").NextUntil("[ng-cloak]")
	assertLength(t, sel.Nodes, 1)
	assertSelectionIs(t, sel, "#cf2-2")
}

func TestNextUntilOrder(t *testing.T) {
	sel := Doc().Find("#cf2-1").NextUntil("#cf2-4")
	assertLength(t, sel.Nodes, 2)
	assertSelectionIs(t, sel, "#cf2-2", "#cf2-3")
}

func TestNextUntilRollback(t *testing.T) {
	sel := Doc().Find("#cf2-1")
	sel2 := sel.PrevUntil("#cf2-4").End()
	assertEqual(t, sel, sel2)
}

func TestNextUntilSelection(t *testing.T) {
	sel := Doc2().Find("#n2")
	sel2 := Doc2().Find("#n4")
	sel2 = sel.NextUntilSelection(sel2)
	assertLength(t, sel2.Nodes, 1)
	assertSelectionIs(t, sel2, "#n3")
}

func TestNextUntilSelectionRollback(t *testing.T) {
	sel := Doc2().Find("#n2")
	sel2 := Doc2().Find("#n4")
	sel2 = sel.NextUntilSelection(sel2).End()
	assertEqual(t, sel, sel2)
}

func TestNextUntilNodes(t *testing.T) {
	sel := Doc2().Find("#n2")
	sel2 := Doc2().Find("#n5")
	sel2 = sel.NextUntilNodes(sel2.Nodes...)
	assertLength(t, sel2.Nodes, 2)
	assertSelectionIs(t, sel2, "#n3", "#n4")
}

func TestNextUntilNodesRollback(t *testing.T) {
	sel := Doc2().Find("#n2")
	sel2 := Doc2().Find("#n5")
	sel2 = sel.NextUntilNodes(sel2.Nodes...).End()
	assertEqual(t, sel, sel2)
}

func TestPrevUntil(t *testing.T) {
	sel := Doc().Find(".alert p").PrevUntil("a")
	assertLength(t, sel.Nodes, 1)
	assertSelectionIs(t, sel, "h4")
}

func TestPrevUntil2(t *testing.T) {
	sel := Doc().Find("[ng-cloak]").PrevUntil(":not([ng-cloak])")
	assertLength(t, sel.Nodes, 1)
	assertSelectionIs(t, sel, "[ng-cloak]")
}

func TestPrevUntilOrder(t *testing.T) {
	sel := Doc().Find("#cf2-4").PrevUntil("#cf2-1")
	assertLength(t, sel.Nodes, 2)
	assertSelectionIs(t, sel, "#cf2-3", "#cf2-2")
}

func TestPrevUntilRollback(t *testing.T) {
	sel := Doc().Find("#cf2-4")
	sel2 := sel.PrevUntil("#cf2-1").End()
	assertEqual(t, sel, sel2)
}

func TestPrevUntilSelection(t *testing.T) {
	sel := Doc2().Find("#n4")
	sel2 := Doc2().Find("#n2")
	sel2 = sel.PrevUntilSelection(sel2)
	assertLength(t, sel2.Nodes, 1)
	assertSelectionIs(t, sel2, "#n3")
}

func TestPrevUntilSelectionRollback(t *testing.T) {
	sel := Doc2().Find("#n4")
	sel2 := Doc2().Find("#n2")
	sel2 = sel.PrevUntilSelection(sel2).End()
	assertEqual(t, sel, sel2)
}

func TestPrevUntilNodes(t *testing.T) {
	sel := Doc2().Find("#n5")
	sel2 := Doc2().Find("#n2")
	sel2 = sel.PrevUntilNodes(sel2.Nodes...)
	assertLength(t, sel2.Nodes, 2)
	assertSelectionIs(t, sel2, "#n4", "#n3")
}

func TestPrevUntilNodesRollback(t *testing.T) {
	sel := Doc2().Find("#n5")
	sel2 := Doc2().Find("#n2")
	sel2 = sel.PrevUntilNodes(sel2.Nodes...).End()
	assertEqual(t, sel, sel2)
}

func TestNextFilteredUntil(t *testing.T) {
	sel := Doc2().Find(".two").NextFilteredUntil(".even", ".six")
	assertLength(t, sel.Nodes, 4)
	assertSelectionIs(t, sel, "#n3", "#n5", "#nf3", "#nf5")
}

func TestNextFilteredUntilRollback(t *testing.T) {
	sel := Doc2().Find(".two")
	sel2 := sel.NextFilteredUntil(".even", ".six").End()
	assertEqual(t, sel, sel2)
}

func TestNextFilteredUntilSelection(t *testing.T) {
	sel := Doc2().Find(".even")
	sel2 := Doc2().Find(".five")
	sel = sel.NextFilteredUntilSelection(".even", sel2)
	assertLength(t, sel.Nodes, 2)
	assertSelectionIs(t, sel, "#n3", "#nf3")
}

func TestNextFilteredUntilSelectionRollback(t *testing.T) {
	sel := Doc2().Find(".even")
	sel2 := Doc2().Find(".five")
	sel3 := sel.NextFilteredUntilSelection(".even", sel2).End()
	assertEqual(t, sel, sel3)
}

func TestNextFilteredUntilNodes(t *testing.T) {
	sel := Doc2().Find(".even")
	sel2 := Doc2().Find(".four")
	sel = sel.NextFilteredUntilNodes(".odd", sel2.Nodes...)
	assertLength(t, sel.Nodes, 4)
	assertSelectionIs(t, sel, "#n2", "#n6", "#nf2", "#nf6")
}

func TestNextFilteredUntilNodesRollback(t *testing.T) {
	sel := Doc2().Find(".even")
	sel2 := Doc2().Find(".four")
	sel3 := sel.NextFilteredUntilNodes(".odd", sel2.Nodes...).End()
	assertEqual(t, sel, sel3)
}

func TestPrevFilteredUntil(t *testing.T) {
	sel := Doc2().Find(".five").PrevFilteredUntil(".odd", ".one")
	assertLength(t, sel.Nodes, 4)
	assertSelectionIs(t, sel, "#n4", "#n2", "#nf4", "#nf2")
}

func TestPrevFilteredUntilRollback(t *testing.T) {
	sel := Doc2().Find(".four")
	sel2 := sel.PrevFilteredUntil(".odd", ".one").End()
	assertEqual(t, sel, sel2)
}

func TestPrevFilteredUntilSelection(t *testing.T) {
	sel := Doc2().Find(".odd")
	sel2 := Doc2().Find(".two")
	sel = sel.PrevFilteredUntilSelection(".odd", sel2)
	assertLength(t, sel.Nodes, 2)
	assertSelectionIs(t, sel, "#n4", "#nf4")
}

func TestPrevFilteredUntilSelectionRollback(t *testing.T) {
	sel := Doc2().Find(".even")
	sel2 := Doc2().Find(".five")
	sel3 := sel.PrevFilteredUntilSelection(".even", sel2).End()
	assertEqual(t, sel, sel3)
}

func TestPrevFilteredUntilNodes(t *testing.T) {
	sel := Doc2().Find(".even")
	sel2 := Doc2().Find(".four")
	sel = sel.PrevFilteredUntilNodes(".odd", sel2.Nodes...)
	assertLength(t, sel.Nodes, 2)
	assertSelectionIs(t, sel, "#n2", "#nf2")
}

func TestPrevFilteredUntilNodesRollback(t *testing.T) {
	sel := Doc2().Find(".even")
	sel2 := Doc2().Find(".four")
	sel3 := sel.PrevFilteredUntilNodes(".odd", sel2.Nodes...).End()
	assertEqual(t, sel, sel3)
}

func TestClosestItself(t *testing.T) {
	sel := Doc2().Find(".three")
	sel2 := sel.Closest(".row")
	assertLength(t, sel2.Nodes, sel.Length())
	assertSelectionIs(t, sel2, "#n3", "#nf3")
}

func TestClosestNoDupes(t *testing.T) {
	sel := Doc().Find(".span12")
	sel2 := sel.Closest(".pvk-content")
	assertLength(t, sel2.Nodes, 1)
	assertClass(t, sel2, "pvk-content")
}

func TestClosestNone(t *testing.T) {
	sel := Doc().Find("h4")
	sel2 := sel.Closest("a")
	assertLength(t, sel2.Nodes, 0)
}

func TestClosestMany(t *testing.T) {
	sel := Doc().Find(".container-fluid")
	sel2 := sel.Closest(".pvk-content")
	assertLength(t, sel2.Nodes, 2)
	assertSelectionIs(t, sel2, "#pc1", "#pc2")
}

func TestClosestRollback(t *testing.T) {
	sel := Doc().Find(".container-fluid")
	sel2 := sel.Closest(".pvk-content").End()
	assertEqual(t, sel, sel2)
}

func TestClosestSelectionItself(t *testing.T) {
	sel := Doc2().Find(".three")
	sel2 := sel.ClosestSelection(Doc2().Find(".row"))
	assertLength(t, sel2.Nodes, sel.Length())
}

func TestClosestSelectionNoDupes(t *testing.T) {
	sel := Doc().Find(".span12")
	sel2 := sel.ClosestSelection(Doc().Find(".pvk-content"))
	assertLength(t, sel2.Nodes, 1)
	assertClass(t, sel2, "pvk-content")
}

func TestClosestSelectionNone(t *testing.T) {
	sel := Doc().Find("h4")
	sel2 := sel.ClosestSelection(Doc().Find("a"))
	assertLength(t, sel2.Nodes, 0)
}

func TestClosestSelectionMany(t *testing.T) {
	sel := Doc().Find(".container-fluid")
	sel2 := sel.ClosestSelection(Doc().Find(".pvk-content"))
	assertLength(t, sel2.Nodes, 2)
	assertSelectionIs(t, sel2, "#pc1", "#pc2")
}

func TestClosestSelectionRollback(t *testing.T) {
	sel := Doc().Find(".container-fluid")
	sel2 := sel.ClosestSelection(Doc().Find(".pvk-content")).End()
	assertEqual(t, sel, sel2)
}

func TestClosestNodesItself(t *testing.T) {
	sel := Doc2().Find(".three")
	sel2 := sel.ClosestNodes(Doc2().Find(".row").Nodes...)
	assertLength(t, sel2.Nodes, sel.Length())
}

func TestClosestNodesNoDupes(t *testing.T) {
	sel := Doc().Find(".span12")
	sel2 := sel.ClosestNodes(Doc().Find(".pvk-content").Nodes...)
	assertLength(t, sel2.Nodes, 1)
	assertClass(t, sel2, "pvk-content")
}

func TestClosestNodesNone(t *testing.T) {
	sel := Doc().Find("h4")
	sel2 := sel.ClosestNodes(Doc().Find("a").Nodes...)
	assertLength(t, sel2.Nodes, 0)
}

func TestClosestNodesMany(t *testing.T) {
	sel := Doc().Find(".container-fluid")
	sel2 := sel.ClosestNodes(Doc().Find(".pvk-content").Nodes...)
	assertLength(t, sel2.Nodes, 2)
	assertSelectionIs(t, sel2, "#pc1", "#pc2")
}

func TestClosestNodesRollback(t *testing.T) {
	sel := Doc().Find(".container-fluid")
	sel2 := sel.ClosestNodes(Doc().Find(".pvk-content").Nodes...).End()
	assertEqual(t, sel, sel2)
}

func TestIssue26(t *testing.T) {
	img1 := `<img src="assets/images/gallery/thumb-1.jpg" alt="150x150" />`
	img2 := `<img alt="150x150" src="assets/images/gallery/thumb-1.jpg" />`
	cases := []struct {
		s string
		l int
	}{
		{s: img1 + img2, l: 2},
		{s: img1, l: 1},
		{s: img2, l: 1},
	}
	for _, c := range cases {
		doc, err := NewDocumentFromReader(strings.NewReader(c.s))
		if err != nil {
			t.Fatal(err)
		}
		sel := doc.Find("img[src]")
		assertLength(t, sel.Nodes, c.l)
	}
}
