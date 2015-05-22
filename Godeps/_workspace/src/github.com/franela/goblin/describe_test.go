package goblin

import (
	"testing"
)

func TestBefore(t *testing.T) {
	fakeTest := testing.T{}

	g := Goblin(&fakeTest)

	g.Describe("Numbers", func() {
		before := 0

		g.Before(func() {
			before++
		})

		g.It("Should have called before", func() {
			g.Assert(before).Equal(1)
		})

		g.It("Should have called before only once", func() {
			g.Assert(before).Equal(1)
		})
	})

	if fakeTest.Failed() {
		t.Fatal()
	}
}

func TestMultipleBefore(t *testing.T) {
	fakeTest := testing.T{}

	g := Goblin(&fakeTest)

	g.Describe("Numbers", func() {
		before := 0

		g.Before(func() {
			before++
		})

		g.Before(func() {
			before++
		})

		g.It("Should have called all the registered before", func() {
			g.Assert(before).Equal(2)
		})
	})

	if fakeTest.Failed() {
		t.Fatal()
	}
}

func TestNestedBefore(t *testing.T) {
	fakeTest := testing.T{}

	g := Goblin(&fakeTest)

	g.Describe("Numbers", func() {
		before := 0

		g.Before(func() {
			before++
		})

		g.Describe("Addition", func() {
			g.Before(func() {
				before++
			})

			g.It("Should have called all the registered before", func() {
				g.Assert(before).Equal(2)
			})

			g.It("Should have called all the registered before only once", func() {
				g.Assert(before).Equal(2)
			})
		})

	})

	if fakeTest.Failed() {
		t.Fatal()
	}
}

func TestAfter(t *testing.T) {
	fakeTest := testing.T{}

	g := Goblin(&fakeTest)
	after := 0
	g.Describe("Numbers", func() {

		g.After(func() {
			after++
		})

		g.It("Should call after only once", func() {
			g.Assert(after).Equal(0)
		})

		g.It("Should call after only once", func() {
			g.Assert(after).Equal(0)
		})
	})

	if fakeTest.Failed() || after != 1 {
		t.Fatal()
	}
}

func TestMultipleAfter(t *testing.T) {
	fakeTest := testing.T{}

	g := Goblin(&fakeTest)

	after := 0
	g.Describe("Numbers", func() {

		g.After(func() {
			after++
		})

		g.After(func() {
			after++
		})

		g.It("Should call all the registered after", func() {
			g.Assert(after).Equal(0)
		})
	})

	if fakeTest.Failed() && after != 2 {
		t.Fatal()
	}
}

func TestNestedAfter(t *testing.T) {
	fakeTest := testing.T{}

	g := Goblin(&fakeTest)
	after := 0
	g.Describe("Numbers", func() {

		g.After(func() {
			after++
		})

		g.Describe("Addition", func() {
			g.After(func() {
				after++
			})

			g.It("Should call all the registered after", func() {
				g.Assert(after).Equal(0)
			})

			g.It("Should have called all the registered after only once", func() {
				g.Assert(after).Equal(0)
			})
		})

	})

	if fakeTest.Failed() || after != 2 {
		t.Fatal()
	}
}
