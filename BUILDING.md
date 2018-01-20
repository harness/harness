```bash
# 1. Install go 1.9 or later

# 2. Install dependencies

go get -u github.com/drone/drone-ui/dist
go get -u golang.org/x/net/context
go get -u golang.org/x/net/context/ctxhttp
go get -u github.com/golang/protobuf/proto
go get -u github.com/golang/protobuf/protoc-gen-go

# 3. Install drone

go get -d github.com/drone/drone/...

# 4. Install binaries to $GOPATH/bin

go install github.com/drone/drone/cmd/drone-agent
go install github.com/drone/drone/cmd/drone-server

# 5. Test

cd $GOPATH/src/github.com/drone/drone
go test -cover $(go list ./... | grep -v /vendor/)

# 6. Build

GOOS=linux GOARCH=amd64  \
    go build -o release/drone-server \
    ./cmd/drone-server

GOOS=linux GOARCH=amd64  \
    go build -o release/drone-agent \
    ./cmd/drone-agent
```
