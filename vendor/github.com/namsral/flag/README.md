Flag
===

Flag is a drop in replacement for Go's flag package with the addition to parse files and environment variables. If you support the [twelve-factor app methodology][], Flag complies with the third factor; "Store config in the environment".

[twelve-factor app methodology]: http://12factor.net

An example using a gopher:

```go
$ cat > gopher.go
    package main

    import (
        "fmt"
    	"github.com/namsral/flag"
	)
    
    var age int
    
    flag.IntVar(&age, "age", 0, "age of gopher")
    flag.Parse()
    
    fmt.Print("age:", age)

$ go run gopher.go -age 1
age: 1
```

Same code but using an environment variable:

```go
$ export AGE=2
$ go run gopher.go
age: 2
```
    

Same code but using a configuration file:

```go
$ cat > gopher.conf
age 3

$ go run gopher.go -config gopher.conf
age: 3
```

The following table shows how flags are translated to environment variables and configuration files:

| Type   | Flag          | Environment  | File         |
| ------ | :------------ |:------------ |:------------ |
| int    | -age 2        | AGE=2        | age 2        |
| bool   | -female       | FEMALE=true  | female true  |
| float  | -length 175.5 | LENGTH=175.5 | length 175.5 |
| string | -name Gloria  | NAME=Gloria  | name Gloria  |

This package is a port of Go's [flag][] package from the standard library with the addition of two functions `ParseEnv` and `ParseFile`.

[flag]: http://golang.org/src/pkg/flagconfiguration


Why?
---

Why not use one of the many INI, JSON or YAML parsers?

I find it best practice to have simple configuration options to control the behaviour of an applications when it starts up. Use basic types like ints, floats and strings for configuration options and store more complex data structures in the "datastore" layer.


Usage
---

It's intended for projects which require a simple configuration made available through command-line flags, configuration files and shell environments. It's similar to the original `flag` package.

Example:

```go
import "github/namsral/flag"

flag.String("config", "", "help message for config")
flag.Int("age", 24, "help message for age")

flag.Parse()
```

Order of precedence:

1. Command line options
2. Environment variables
3. Configuration file
4. Default values


#### Parsing Configuration Files

Create a configuration file:

```go
$ cat > ./gopher.conf
# empty newlines and lines beginning with a "#" character are ignored.
name bob

# keys and values can also be separated by the "=" character
age=20

# booleans can be empty, set with 0, 1, true, false, etc
hacker
```

Add a "config" flag:

```go
flag.String("config", "", "help message for config")
```

Run the command:

```go
$ go run ./gopher.go -config ./gopher.conf
```

#### Parsing Environment Variables

Environment variables are parsed 1-on-1 with defined flags:

```go
$ export AGE=44
$ go run ./gopher.go
age=44
```


You can also parse prefixed environment variables by setting a prefix name when creating a new empty flag set:

```go
fs := flag.NewFlagSetWithEnvPrefix(os.Args[0], "GO", 0)
fs.Int("age", 24, "help message for age")
fs.Parse(os.Args[1:])
...
$ go export GO_AGE=33
$ go run ./gopher.go
age=33
```


For more examples see the [examples][] directory in the project repository.

[examples]: https://github.com/namsral/flag/tree/master/examples

That's it.


License
---


Copyright (c) 2012 The Go Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
