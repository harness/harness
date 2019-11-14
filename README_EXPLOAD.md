# Local Building

```
go build github.com/drone/drone/cmd/drone-server
```

# Building and pushing the Docker image

```
docker build -t gcr.io/time-coin/drone-server-custom:1.6.1.14 .
docker push gcr.io/time-coin/drone-server-custom:1.6.1.14
```

