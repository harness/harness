package goblin

import (
	"testing"
)

func TestBeforeEach(t *testing.T) {
	fakeTest := testing.T{}

	g := Goblin(&fakeTest)

	g.Describe("Numbers", func() {
		before := 0

		g.BeforeEach(func() {
			before++
		})

		g.It("Should have called beforeEach", func() {
			g.Assert(before).Equal(1)
		})

		g.It("Should have called beforeEach also for this one", func() {
			g.Assert(before).Equal(2)
		})
	})

	if fakeTest.Failed() {
		t.Fatal()
	}
}

func TestMultipleBeforeEach(t *testing.T) {
	fakeTest := testing.T{}

	g := Goblin(&fakeTest)

	g.Describe("Numbers", func() {
		before := 0

		g.BeforeEach(func() {
			before++
		})

		g.BeforeEach(func() {
			before++
		})

		g.It("Should have called all the registered beforeEach", func() {
			g.Assert(before).Equal(2)
		})
	})

	if fakeTest.Failed() {
		t.Fatal()
	}
}

func TestNestedBeforeEach(t *testing.T) {
	fakeTest := testing.T{}

	g := Goblin(&fakeTest)

	g.Describe("Numbers", func() {
		before := 0

		g.BeforeEach(func() {
			before++
		})

		g.Describe("Addition", func() {
			g.BeforeEach(func() {
				before++
			})

			g.It("Should have called all the registered beforeEach", func() {
				g.Assert(before).Equal(2)
			})

			g.It("Should have called all the registered beforeEach also for this one", func() {
				g.Assert(before).Equal(4)
			})
		})

	})

	if fakeTest.Failed() {
		t.Fatal()
	}
}

func TestAfterEach(t *testing.T) {
	fakeTest := testing.T{}
	after := 0

	g := Goblin(&fakeTest)
	g.Describe("Numbers", func() {

		g.AfterEach(func() {
			after++
		})

		g.It("Should call afterEach after this test", func() {
			g.Assert(after).Equal(0)
		})

		g.It("Should have called afterEach before this test ", func() {
			g.Assert(after).Equal(1)
		})
	})

	if fakeTest.Failed() || after != 2 {
		t.Fatal()
	}
}

func TestMultipleAfterEach(t *testing.T) {
	fakeTest := testing.T{}

	g := Goblin(&fakeTest)

	after := 0
	g.Describe("Numbers", func() {

		g.AfterEach(func() {
			after++
		})

		g.AfterEach(func() {
			after++
		})

		g.It("Should call all the registered afterEach", func() {
			g.Assert(after).Equal(0)
		})
	})

	if fakeTest.Failed() || after != 2 {
		t.Fatal()
	}
}

func TestNestedAfterEach(t *testing.T) {
	fakeTest := testing.T{}

	g := Goblin(&fakeTest)
	after := 0

	g.Describe("Numbers", func() {

		g.AfterEach(func() {
			after++
		})

		g.Describe("Addition", func() {
			g.AfterEach(func() {
				after++
			})

			g.It("Should call all the registered afterEach", func() {
				g.Assert(after).Equal(0)
			})

			g.It("Should have called all the registered aftearEach", func() {
				g.Assert(after).Equal(2)
			})
		})

	})

	if fakeTest.Failed() || after != 4 {
		t.Fatal()
	}
}
