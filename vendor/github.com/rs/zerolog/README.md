# Zero Allocation JSON Logger

[![godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/rs/zerolog) [![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/rs/zerolog/master/LICENSE) [![Build Status](https://travis-ci.org/rs/zerolog.svg?branch=master)](https://travis-ci.org/rs/zerolog) [![Coverage](http://gocover.io/_badge/github.com/rs/zerolog)](http://gocover.io/github.com/rs/zerolog)

The zerolog package provides a fast and simple logger dedicated to JSON output.

Zerolog's API is designed to provide both a great developer experience and stunning [performance](#performance). Its unique chaining API allows zerolog to write JSON log events by avoiding allocations and reflection.

The uber's [zap](https://godoc.org/go.uber.org/zap) library pioneered this approach. Zerolog is taking this concept to the next level with simpler to use API and even better performance.

To keep the code base and the API simple, zerolog focuses on JSON logging only. As [suggested on reddit](https://www.reddit.com/r/golang/comments/6c9k7n/zerolog_is_now_faster_than_zap/), you may use tools like [humanlog](https://github.com/aybabtme/humanlog) to pretty print JSON on the console during development.


## Features

* Level logging
* Sampling
* Contextual fields
* `context.Context` integration
* `net/http` helpers

## Usage

```go
import "github.com/rs/zerolog/log"
```

### A global logger can be use for simple logging

```go
log.Info().Msg("hello world")

// Output: {"level":"info","time":1494567715,"message":"hello world"}
```

NOTE: To import the global logger, import the `log` subpackage `github.com/rs/zerolog/log`.

```go
log.Fatal().
    Err(err).
    Str("service", service).
    Msgf("Cannot start %s", service)

// Output: {"level":"fatal","time":1494567715,"message":"Cannot start myservice","error":"some error","service":"myservice"}
// Exit 1
```

NOTE: Using `Msgf` generates one allocation even when the logger is disabled.

### Fields can be added to log messages

```go
log.Info().
    Str("foo", "bar").
    Int("n", 123).
    Msg("hello world")

// Output: {"level":"info","time":1494567715,"foo":"bar","n":123,"message":"hello world"}
```

### Create logger instance to manage different outputs

```go
logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

logger.Info().Str("foo", "bar").Msg("hello world")

// Output: {"level":"info","time":1494567715,"message":"hello world","foo":"bar"}
```

### Sub-loggers let you chain loggers with additional context

```go
sublogger := log.With().
                 Str("component": "foo").
                 Logger()
sublogger.Info().Msg("hello world")

// Output: {"level":"info","time":1494567715,"message":"hello world","component":"foo"}
```

### Level logging

```go
zerolog.SetGlobalLevel(zerolog.InfoLevel)

log.Debug().Msg("filtered out message")
log.Info().Msg("routed message")

if e := log.Debug(); e.Enabled() {
    // Compute log output only if enabled.
    value := compute()
    e.Str("foo": value).Msg("some debug message")
}

// Output: {"level":"info","time":1494567715,"message":"routed message"}
```

### Sub dictionary

```go
log.Info().
    Str("foo", "bar").
    Dict("dict", zerolog.Dict().
        Str("bar", "baz").
        Int("n", 1)
    ).Msg("hello world")

// Output: {"level":"info","time":1494567715,"foo":"bar","dict":{"bar":"baz","n":1},"message":"hello world"}
```

### Customize automatic field names

```go
zerolog.TimestampFieldName = "t"
zerolog.LevelFieldName = "l"
zerolog.MessageFieldName = "m"

log.Info().Msg("hello world")

// Output: {"l":"info","t":1494567715,"m":"hello world"}
```

### Log with no level nor message

```go
log.Log().Str("foo","bar").Msg("")

// Output: {"time":1494567715,"foo":"bar"}
```

### Add contextual fields to the global logger

```go
log.Logger = log.With().Str("foo", "bar").Logger()
```

### Log Sampling

```go
sampled := log.Sample(10)
sampled.Info().Msg("will be logged every 10 messages")

// Output: {"time":1494567715,"sample":10,"message":"will be logged every 10 messages"}
```

### Pass a sub-logger by context

```go
ctx := log.With("component", "module").Logger().WithContext(ctx)

log.Ctx(ctx).Info().Msg("hello world")

// Output: {"component":"module","level":"info","message":"hello world"}
```

### Set as standard logger output

```go
log := zerolog.New(os.Stdout).With().
    Str("foo", "bar").
    Logger()

stdlog.SetFlags(0)
stdlog.SetOutput(log)

stdlog.Print("hello world")

// Output: {"foo":"bar","message":"hello world"}
```

### Integration with `net/http`

The `github.com/rs/zerolog/hlog` package provides some helpers to integrate zerolog with `http.Handler`.

In this example we use [alice](https://github.com/justinas/alice) to install logger for better readability.

```go
log := zerolog.New(os.Stdout).With().
    Timestamp().
    Str("role", "my-service").
    Str("host", host).
    Logger()

c := alice.New()

// Install the logger handler with default output on the console
c = c.Append(hlog.NewHandler(log))

// Install some provided extra handler to set some request's context fields.
// Thanks to those handler, all our logs will come with some pre-populated fields.
c = c.Append(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
    hlog.FromRequest(r).Info().
        Str("method", r.Method).
        Str("url", r.URL.String()).
        Int("status", status).
        Int("size", size).
        Dur("duration", duration).
        Msg("")
}))
c = c.Append(hlog.RemoteAddrHandler("ip"))
c = c.Append(hlog.UserAgentHandler("user_agent"))
c = c.Append(hlog.RefererHandler("referer"))
c = c.Append(hlog.RequestIDHandler("req_id", "Request-Id"))

// Here is your final handler
h := c.Then(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // Get the logger from the request's context. You can safely assume it
    // will be always there: if the handler is removed, hlog.FromRequest
    // will return a no-op logger.
    hlog.FromRequest(r).Info().
        Str("user", "current user").
        Str("status", "ok").
        Msg("Something happened")

    // Output: {"level":"info","time":"2001-02-03T04:05:06Z","role":"my-service","host":"local-hostname","req_id":"b4g0l5t6tfid6dtrapu0","user":"current user","status":"ok","message":"Something happened"}
}))
http.Handle("/", h)

if err := http.ListenAndServe(":8080", nil); err != nil {
    log.Fatal().Err(err).Msg("Startup failed")
}
```

## Global Settings

Some settings can be changed and will by applied to all loggers:

* `log.Logger`: You can set this value to customize the global logger (the one used by package level methods).
* `zerolog.SetGlobalLevel`: Can raise the minimum level of all loggers. Set this to `zerolog.Disable` to disable logging altogether (quiet mode).
* `zerolog.DisableSampling`: If argument is `true`, all sampled loggers will stop sampling and issue 100% of their log events.
* `zerolog.TimestampFieldName`: Can be set to customize `Timestamp` field name.
* `zerolog.LevelFieldName`: Can be set to customize level field name.
* `zerolog.MessageFieldName`: Can be set to customize message field name.
* `zerolog.ErrorFieldName`: Can be set to customize `Err` field name.
* `zerolog.SampleFieldName`: Can be set to customize the field name added when sampling is enabled.
* `zerolog.TimeFieldFormat`: Can be set to customize `Time` field value formatting. If set with an empty string, times are formated as UNIX timestamp.
	// DurationFieldUnit defines the unit for time.Duration type fields added
	// using the Dur method.
* `DurationFieldUnit`: Sets the unit of the fields added by `Dur` (default: `time.Millisecond`).
* `DurationFieldInteger`: If set to true, `Dur` fields are formatted as integers instead of floats.

## Field Types

### Standard Types

* `Str`
* `Bool`
* `Int`, `Int8`, `Int16`, `Int32`, `Int64`
* `Uint`, `Uint8`, `Uint16`, `Uint32`, `Uint64`
* `Float32`, `Float64`

### Advanced Fields

* `Err`: Takes an `error` and render it as a string using the `zerolog.ErrorFieldName` field name.
* `Timestamp`: Insert a timestamp field with `zerolog.TimestampFieldName` field name and formatted using `zerolog.TimeFieldFormat`.
* `Time`: Adds a field with the time formated with the `zerolog.TimeFieldFormat`.
* `Dur`: Adds a field with a `time.Duration`.
* `Dict`: Adds a sub-key/value as a field of the event.
* `Interface`: Uses reflection to marshal the type.

## Performance

All operations are allocation free (those numbers *include* JSON encoding):

```
BenchmarkLogEmpty-8        100000000    19.1 ns/op	   0 B/op       0 allocs/op
BenchmarkDisabled-8        500000000     4.07 ns/op	   0 B/op       0 allocs/op
BenchmarkInfo-8            30000000	    42.5 ns/op	   0 B/op       0 allocs/op
BenchmarkContextFields-8   30000000	    44.9 ns/op	   0 B/op       0 allocs/op
BenchmarkLogFields-8       10000000	   184 ns/op	   0 B/op       0 allocs/op
```

Using Uber's zap [comparison benchmark](https://github.com/uber-go/zap#performance):

Log a message and 10 fields:

| Library | Time | Bytes Allocated | Objects Allocated |
| :--- | :---: | :---: | :---: |
| zerolog | 767 ns/op | 552 B/op | 6 allocs/op |
| :zap: zap | 848 ns/op | 704 B/op | 2 allocs/op |
| :zap: zap (sugared) | 1363 ns/op | 1610 B/op | 20 allocs/op |
| go-kit | 3614 ns/op | 2895 B/op | 66 allocs/op |
| lion | 5392 ns/op | 5807 B/op | 63 allocs/op |
| logrus | 5661 ns/op | 6092 B/op | 78 allocs/op |
| apex/log | 15332 ns/op | 3832 B/op | 65 allocs/op |
| log15 | 20657 ns/op | 5632 B/op | 93 allocs/op |

Log a message with a logger that already has 10 fields of context:

| Library | Time | Bytes Allocated | Objects Allocated |
| :--- | :---: | :---: | :---: |
| zerolog | 52 ns/op | 0 B/op | 0 allocs/op |
| :zap: zap | 283 ns/op | 0 B/op | 0 allocs/op |
| :zap: zap (sugared) | 337 ns/op | 80 B/op | 2 allocs/op |
| lion | 2702 ns/op | 4074 B/op | 38 allocs/op |
| go-kit | 3378 ns/op | 3046 B/op | 52 allocs/op |
| logrus | 4309 ns/op | 4564 B/op | 63 allocs/op |
| apex/log | 13456 ns/op | 2898 B/op | 51 allocs/op |
| log15 | 14179 ns/op | 2642 B/op | 44 allocs/op |

Log a static string, without any context or `printf`-style templating:

| Library | Time | Bytes Allocated | Objects Allocated |
| :--- | :---: | :---: | :---: |
| zerolog | 50 ns/op | 0 B/op | 0 allocs/op |
| :zap: zap | 236 ns/op | 0 B/op | 0 allocs/op |
| standard library | 453 ns/op | 80 B/op | 2 allocs/op |
| :zap: zap (sugared) | 337 ns/op | 80 B/op | 2 allocs/op |
| go-kit | 508 ns/op | 656 B/op | 13 allocs/op |
| lion | 771 ns/op | 1224 B/op | 10 allocs/op |
| logrus | 1244 ns/op | 1505 B/op | 27 allocs/op |
| apex/log | 2751 ns/op | 584 B/op | 11 allocs/op |
| log15 | 5181 ns/op | 1592 B/op | 26 allocs/op |

