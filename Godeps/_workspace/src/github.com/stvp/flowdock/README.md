flowdock
========

A Flowdock client for Golang. See the [Flowdock API docs][api_docs] for more
information on the individual attributes.

[Go API Documentation][godocs]

Examples
--------

You can set global defaults and use `flowdock.Inbox()` directly:

```go
import (
  "github.com/stvp/flowdock"
)

func main() {
  flowdock.Token = "732da505d0284d5b1d909b1a32426345"
  flowdock.Source = "My Cool App"
  flowdock.FromAddress = "cool@dudes.com"
  // See API docs for more variables

  go flowdock.Inbox("My subject", "My content goes here.")
}
```

Or you can create a `flowdock.Client` to use different Flowdock message
settings in the same app:

```go
import (
  "github.com/stvp/flowdock"
)

func main() {
  client := flowdock.Client{
    Token:       "732da505d0284d5b1d909b1a32426345",
    Source:      "App A",
    FromAddress: "email@stovepipestudios.com",
    FromName:    "Client A",
    ReplyTo:     "app_a@stovepipestudios.com",
    Tags:        []string{"app_a"},
  }

  go client.Inbox("Subject", "Content")
}
```

[api_docs]: https://www.flowdock.com/api/team-inbox
[godocs]: http://godoc.org/github.com/stvp/flowdock

