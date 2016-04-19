atime 
==========

[![GoDoc](https://godoc.org/github.com/djherbis/atime?status.svg)](https://godoc.org/github.com/djherbis/atime)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg)](LICENSE.txt)
[![Build Status](https://travis-ci.org/djherbis/atime.svg?branch=master)](https://travis-ci.org/djherbis/atime)
[![Coverage Status](https://coveralls.io/repos/djherbis/atime/badge.svg?branch=master)](https://coveralls.io/r/djherbis/atime?branch=master)

Usage
------------
File Access Times for #golang

Looking for ctime or btime? Checkout https://github.com/djherbis/times

Go has a hidden atime function for most platforms, this repo makes it accessible.

```go
package main

import (
  "log"

  "github.com/djherbis/atime"
)

func main() {
  at, err := atime.Stat("myfile")
  if err != nil {
    log.Fatal(err.Error())
  }
  log.Println(at)
}
```

Installation
------------
```sh
go get github.com/djherbis/atime
```
