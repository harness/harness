$env:CGO_ENABLED="0"
go build -o release/windows/amd64/drone-agent github.com/drone/drone/cmd/drone-agent
