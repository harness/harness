# ABool :bulb:
[![Go Report Card](https://goreportcard.com/badge/github.com/tevino/abool)](https://goreportcard.com/report/github.com/tevino/abool)
[![GoDoc](https://godoc.org/github.com/tevino/abool?status.svg)](https://godoc.org/github.com/tevino/abool)

Atomic Boolean library for Golang, optimized for performance yet simple to use.

Use this for cleaner code.

## Usage

```go
import "github.com/tevino/abool"

cond := abool.New()  // default to false

cond.Set()                 // Set to true
cond.IsSet()               // Returns true
cond.UnSet()               // Set to false
cond.SetTo(true)           // Set to whatever you want
cond.SetToIf(false, true)  // Set to true if it is false, returns false(not set)


// embedding
type Foo struct {
    cond *abool.AtomicBool  // always use pointer to avoid copy
}
```

## Benchmark:

- Golang 1.6.2
- OS X 10.11.4

```shell
# Read
BenchmarkMutexRead-4       	100000000	        21.0 ns/op
BenchmarkAtomicValueRead-4 	200000000	         6.30 ns/op
BenchmarkAtomicBoolRead-4  	300000000	         4.21 ns/op  # <--- This package

# Write
BenchmarkMutexWrite-4      	100000000	        21.6 ns/op
BenchmarkAtomicValueWrite-4	 30000000	        43.4 ns/op
BenchmarkAtomicBoolWrite-4 	200000000	         9.87 ns/op  # <--- This package

# CAS
BenchmarkMutexCAS-4        	 30000000	        44.9 ns/op
BenchmarkAtomicBoolCAS-4   	100000000	        11.7 ns/op   # <--- This package
```

