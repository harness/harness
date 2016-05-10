package goquery

import (
	"testing"
)

func TestFirst(t *testing.T) {
	sel := Doc().Find(".pvk-content").First()
	assertLength(t, sel.Nodes, 1)
}

func TestFirstEmpty(t *testing.T) {
	sel := Doc().Find(".pvk-zzcontentzz").First()
	assertLength(t, sel.Nodes, 0)
}

func TestFirstRollback(t *testing.T) {
	sel := Doc().Find(".pvk-content")
	sel2 := sel.First().End()
	assertEqual(t, sel, sel2)
}

func TestLast(t *testing.T) {
	sel := Doc().Find(".pvk-content").Last()
	assertLength(t, sel.Nodes, 1)

	// Should contain Footer
	foot := Doc().Find(".footer")
	if !sel.Contains(foot.Nodes[0]) {
		t.Error("Last .pvk-content should contain .footer.")
	}
}

func TestLastEmpty(t *testing.T) {
	sel := Doc().Find(".pvk-zzcontentzz").Last()
	assertLength(t, sel.Nodes, 0)
}

func TestLastRollback(t *testing.T) {
	sel := Doc().Find(".pvk-content")
	sel2 := sel.Last().End()
	assertEqual(t, sel, sel2)
}

func TestEq(t *testing.T) {
	sel := Doc().Find(".pvk-content").Eq(1)
	assertLength(t, sel.Nodes, 1)
}

func TestEqNegative(t *testing.T) {
	sel := Doc().Find(".pvk-content").Eq(-1)
	assertLength(t, sel.Nodes, 1)

	// Should contain Footer
	foot := Doc().Find(".footer")
	if !sel.Contains(foot.Nodes[0]) {
		t.Error("Index -1 of .pvk-content should contain .footer.")
	}
}

func TestEqEmpty(t *testing.T) {
	sel := Doc().Find("something_random_that_does_not_exists").Eq(0)
	assertLength(t, sel.Nodes, 0)
}

func TestEqInvalidPositive(t *testing.T) {
	sel := Doc().Find(".pvk-content").Eq(3)
	assertLength(t, sel.Nodes, 0)
}

func TestEqInvalidNegative(t *testing.T) {
	sel := Doc().Find(".pvk-content").Eq(-4)
	assertLength(t, sel.Nodes, 0)
}

func TestEqRollback(t *testing.T) {
	sel := Doc().Find(".pvk-content")
	sel2 := sel.Eq(1).End()
	assertEqual(t, sel, sel2)
}

func TestSlice(t *testing.T) {
	sel := Doc().Find(".pvk-content").Slice(0, 2)

	assertLength(t, sel.Nodes, 2)
}

func TestSliceOutOfBounds(t *testing.T) {
	defer assertPanic(t)
	Doc().Find(".pvk-content").Slice(2, 12)
}

func TestNegativeSliceStart(t *testing.T) {
	sel := Doc().Find(".container-fluid").Slice(-2, 3)
	assertLength(t, sel.Nodes, 1)
	assertSelectionIs(t, sel.Eq(0), "#cf3")
}

func TestNegativeSliceEnd(t *testing.T) {
	sel := Doc().Find(".container-fluid").Slice(1, -1)
	assertLength(t, sel.Nodes, 2)
	assertSelectionIs(t, sel.Eq(0), "#cf2")
	assertSelectionIs(t, sel.Eq(1), "#cf3")
}

func TestNegativeSliceBoth(t *testing.T) {
	sel := Doc().Find(".container-fluid").Slice(-3, -1)
	assertLength(t, sel.Nodes, 2)
	assertSelectionIs(t, sel.Eq(0), "#cf2")
	assertSelectionIs(t, sel.Eq(1), "#cf3")
}

func TestNegativeSliceOutOfBounds(t *testing.T) {
	defer assertPanic(t)
	Doc().Find(".container-fluid").Slice(-12, -7)
}

func TestSliceRollback(t *testing.T) {
	sel := Doc().Find(".pvk-content")
	sel2 := sel.Slice(0, 2).End()
	assertEqual(t, sel, sel2)
}

func TestGet(t *testing.T) {
	sel := Doc().Find(".pvk-content")
	node := sel.Get(1)
	if sel.Nodes[1] != node {
		t.Errorf("Expected node %v to be %v.", node, sel.Nodes[1])
	}
}

func TestGetNegative(t *testing.T) {
	sel := Doc().Find(".pvk-content")
	node := sel.Get(-3)
	if sel.Nodes[0] != node {
		t.Errorf("Expected node %v to be %v.", node, sel.Nodes[0])
	}
}

func TestGetInvalid(t *testing.T) {
	defer assertPanic(t)
	sel := Doc().Find(".pvk-content")
	sel.Get(129)
}

func TestIndex(t *testing.T) {
	sel := Doc().Find(".pvk-content")
	if i := sel.Index(); i != 1 {
		t.Errorf("Expected index of 1, got %v.", i)
	}
}

func TestIndexSelector(t *testing.T) {
	sel := Doc().Find(".hero-unit")
	if i := sel.IndexSelector("div"); i != 4 {
		t.Errorf("Expected index of 4, got %v.", i)
	}
}

func TestIndexOfNode(t *testing.T) {
	sel := Doc().Find("div.pvk-gutter")
	if i := sel.IndexOfNode(sel.Nodes[1]); i != 1 {
		t.Errorf("Expected index of 1, got %v.", i)
	}
}

func TestIndexOfNilNode(t *testing.T) {
	sel := Doc().Find("div.pvk-gutter")
	if i := sel.IndexOfNode(nil); i != -1 {
		t.Errorf("Expected index of -1, got %v.", i)
	}
}

func TestIndexOfSelection(t *testing.T) {
	sel := Doc().Find("div")
	sel2 := Doc().Find(".hero-unit")
	if i := sel.IndexOfSelection(sel2); i != 4 {
		t.Errorf("Expected index of 4, got %v.", i)
	}
}
