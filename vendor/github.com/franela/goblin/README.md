[![Build Status](https://travis-ci.org/franela/goblin.png?branch=master)](https://travis-ci.org/franela/goblin)
Goblin
======

![](https://github.com/marcosnils/goblin/blob/master/goblin_logo.jpg?raw=true)

A [Mocha](http://visionmedia.github.io/mocha/) like BDD testing framework for Go

No extensive documentation nor complicated steps to get it running

Run tests as usual with `go test`

Colorful reports and beautiful syntax


Why Goblin?
-----------

Inspired by the flexibility and simplicity of Node BDD and frustrated by the
rigorousness of Go way of testing, we wanted to bring a new tool to 
write self-describing and comprehensive code.



What do I get with it?
----------------------

- Preserve the exact same syntax and behaviour as Node's Mocha
- Nest as many `Describe` and `It` blocks as you want
- Use `Before`, `BeforeEach`, `After` and `AfterEach` for setup and teardown your tests
- No need to remember confusing parameters in `Describe` and `It` blocks
- Use a declarative and expressive language to write your tests
- Plug different assertion libraries ([Gomega](https://github.com/onsi/gomega) supported so far)
- Skip your tests the same way as you would do in Mocha
- Automatic terminal support for colored outputs
- Two line setup is all you need to get up running



How do I use it?
----------------

Since ```go test``` is not currently extensive, you will have to hook Goblin to it. You do that by
adding a single test method in your test file. All your goblin tests will be implemented inside this function.

```go
package foobar

import (
    "testing"
    . "github.com/franela/goblin"
)

func Test(t *testing.T) {
    g := Goblin(t)
    g.Describe("Numbers", func() {
        g.It("Should add two numbers ", func() {
            g.Assert(1+1).Equal(2)
        })
        g.It("Should match equal numbers", func() {
            g.Assert(2).Equal(4)
        })
        g.It("Should substract two numbers")
    })
}
```

Ouput will be something like:

![](https://github.com/marcosnils/goblin/blob/master/goblin_output.png?raw=true)

Nice and easy, right?

Can I do asynchronous tests?
----------------------------

Yes! Goblin will help you to test asynchronous things, like goroutines, etc. You just need to add a ```done``` parameter to the handler function of your ```It```. This handler function should be called when your test passes.

```go
  ...
  g.Describe("Numbers", func() {
      g.It("Should add two numbers asynchronously", func(done Done) {
          go func() {
              g.Assert(1+1).Equal(2)
              done()
          }()
      })
  })
  ...
```

Goblin will wait for the ```done``` call, a ```Fail``` call or any false assertion.

How do I use it with Gomega?
----------------------------

Gomega is a nice assertion framework. But it doesn't provide a nice way to hook it to testing frameworks. It should just panic instead of requiring a fail function. There is an issue about that [here](https://github.com/onsi/gomega/issues/5).
While this is being discussed and hopefully fixed, the way to use Gomega with Goblin is:

```go
package foobar

import (
    "testing"
    . "github.com/franela/goblin"
    . "github.com/onsi/gomega"
)

func Test(t *testing.T) {
    g := Goblin(t)

    //special hook for gomega
    RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

    g.Describe("lala", func() {
        g.It("lslslslsls", func() {
            Expect(1).To(Equal(10))
        })
    })
}
```


FAQ:
----

### How do I run specific tests?

If `-goblin.run=$REGES` is supplied to the `go test` command then only tests that match the supplied regex will run


TODO:
-----

We do have a couple of [issues](https://github.com/franela/goblin/issues) pending we'll be addressing soon. But feel free to
contribute and send us PRs (with tests please :smile:).

Contributions:
------------

Special thanks to [Leandro Reox](https://github.com/leandroreox) (Leitan) for the goblin logo.
