package goblin

import (
	"testing"
	"time"
)

func TestAddNumbersSucceed(t *testing.T) {
	fakeTest := testing.T{}
	g := Goblin(&fakeTest)

	g.Describe("Numbers", func() {
		g.It("Should add numbers", func() {
			sum := 1 + 1
			g.Assert(sum).Equal(2)
		})
	})

	if fakeTest.Failed() {
		t.Fatal()
	}
}

func TestAddNumbersFails(t *testing.T) {
	fakeTest := testing.T{}

	g := Goblin(&fakeTest)

	g.Describe("Numbers", func() {
		g.It("Should add numbers", func() {
			sum := 1 + 1
			g.Assert(sum).Equal(4)
		})
	})

	if !fakeTest.Failed() {
		t.Fatal()
	}
}

func TestMultipleIts(t *testing.T) {
	fakeTest := testing.T{}

	g := Goblin(&fakeTest)

	count := 0
	g.Describe("Numbers", func() {
		g.It("Should add numbers", func() {
			count++
			sum := 1 + 1
			g.Assert(sum).Equal(2)
		})

		g.It("Should add numbers", func() {
			count++
			sum := 1 + 1
			g.Assert(sum).Equal(4)
		})
	})

	if count != 2 {
		t.Fatal()
	}
}

func TestMultipleDescribes(t *testing.T) {
	fakeTest := testing.T{}

	g := Goblin(&fakeTest)

	count := 0
	g.Describe("Numbers", func() {

		g.Describe("Addition", func() {
			g.It("Should add numbers", func() {
				count++
				sum := 1 + 1
				g.Assert(sum).Equal(2)
			})
		})

		g.Describe("Substraction", func() {
			g.It("Should substract numbers ", func() {
				count++
				sub := 5 - 5
				g.Assert(sub).Equal(1)
			})
		})
	})

	if count != 2 {
		t.Fatal()
	}
}

func TestPending(t *testing.T) {
	fakeTest := testing.T{}

	g := Goblin(&fakeTest)

	g.Describe("Numbers", func() {

		g.It("Should add numbers")

		g.Describe("Substraction", func() {
			g.It("Should substract numbers")
		})

	})

	if fakeTest.Failed() {
		t.Fatal()
	}
}

func TestNotRunBeforesOrAfters(t *testing.T) {
	fakeTest := testing.T{}

	g := Goblin(&fakeTest)
	var count int

	g.Describe("Numbers", func() {
		g.Before(func() {
			count++
		})
		g.BeforeEach(func() {
			count++
		})

		g.After(func() {
			count++
		})
		g.AfterEach(func() {
			count++
		})

		g.Describe("Letters", func() {
			g.Before(func() {
				count++
			})
			g.BeforeEach(func() {
				count++
			})

			g.After(func() {
				count++
			})
			g.AfterEach(func() {
				count++
			})
		})
	})

	if count != 0 {
		t.Fatal()
	}
}

func TestFailOnError(t *testing.T) {
	fakeTest := testing.T{}

	g := Goblin(&fakeTest)

	g.Describe("Numbers", func() {
		g.It("Does something", func() {
			g.Fail("Something")
		})
	})

	g.Describe("Errors", func() {
		g.It("Should fail with structs ", func() {
			var s struct{ error string }
			s.error = "Error"
			g.Fail(s)
		})
	})

	if !fakeTest.Failed() {
		t.Fatal()
	}
}

func TestFailImmediately(t *testing.T) {
	fakeTest := testing.T{}
	g := Goblin(&fakeTest)
	reached := false
	g.Describe("Errors", func() {
		g.It("Should fail immediately for sync test ", func() {
			g.Assert(false).IsTrue()
			reached = true
			g.Assert("foo").Equal("bar")
		})
		g.It("Should fail immediately for async test ", func(done Done) {
			go func() {
				g.Assert(false).IsTrue()
				reached = true
				g.Assert("foo").Equal("bar")
				done()
			}()
		})
	})

	if reached {
		t.Fatal()
	}
}

func TestAsync(t *testing.T) {
	fakeTest := testing.T{}
	g := Goblin(&fakeTest)

	g.Describe("Async test", func() {
		g.It("Should fail when Fail is called immediately", func(done Done) {
			g.Fail("Failed")
		})
		g.It("Should fail when fail is called", func(done Done) {
			go func() {
				time.Sleep(100 * time.Millisecond)
				g.Fail("foo is not bar")
			}()
		})

		g.It("Should fail if done receives a parameter ", func(done Done) {
			go func() {
				time.Sleep(100 * time.Millisecond)
				done("Error")
			}()
		})

		g.It("Should pass when done is called", func(done Done) {
			go func() {
				time.Sleep(100 * time.Millisecond)
				done()
			}()
		})

		g.It("Should fail if done has been called multiple times", func(done Done) {
			go func() {
				time.Sleep(100 * time.Millisecond)
				done()
				done()
			}()
		})
	})

	if !fakeTest.Failed() {
		t.Fatal()
	}
}

func TestTimeout(t *testing.T) {
	fakeTest := testing.T{}
	g := Goblin(&fakeTest, "-goblin.timeout=10ms")

	g.Describe("Test", func() {
		g.It("Should fail if test exceeds the specified timeout with sync test", func() {
			time.Sleep(100 * time.Millisecond)
		})

		g.It("Should fail if test exceeds the specified timeout with async test", func(done Done) {
			time.Sleep(100 * time.Millisecond)
			done()
		})
	})

	if !fakeTest.Failed() {
		t.Fatal()
	}
}
