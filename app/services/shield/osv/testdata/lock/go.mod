module example.com/app

go 1.22

require (
	golang.org/x/text v0.3.6
	github.com/pkg/errors v0.9.1 // indirect
)

require github.com/gin-gonic/gin v1.7.0

require (
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519 // indirect
	github.com/foo/bar v2.0.0+incompatible
)

replace github.com/foo/bar => ../bar
