# Plugin Tutorial

We're going to create a plugin that allows Drone to send a webhook when a build completes. The webhook will send an http POST request to one or more URLs with the build details, encoded in JSON format.

## Plugin Yaml

Many plugins call for some options the user can set, which can be specified in the `.drone.yml` file. This is an example configuration for our `webhook` plugin:

```yaml
---
notify:
  webhook:
    image: bradrydzewski/webhook
    urls:
      - http://foo.com/post/to/this/url
```

The `image` attribute refers to the name of our plugin's Docker image. The `urls` attribute is a custom option allowing the user to define the list of webhook urls.

## Plugin executable

The next step is to write the code for our plugin. Our plugin will need access to information about the running build, and it will need its configuration options. Luckily, the build and repository details are passed to the program as a JSON encoded string over `arg[1]`.

The Drone team created the `drone-plugin-go` library to simplify plugin development, and we'll use this library to create our webhook plugin. First, we'll need to download the library using `go get` and import into our `main.go` file:

```go
import (
    "github.com/drone/drone-plugin-go"
)
```

Next we'll need to read and unmarshal the repository and build detail from `arg[1]`. Our plugin package provides simple helper functions, modeled after the `flag` package, to fetch this data. We'll also need to read and unmarshal our plugin options. Plugin options are passed as the `vargs` paramter.

```go
func main() {
    var repo = plugin.Repo{}
    var build = plugin.Build{}
    var vargs = struct {
        Urls []string `json:"urls"`
    }{}

    plugin.Param("repo", &repo)
    plugin.Param("build", &build)
    plugin.Param("vargs", &vargs)
    plugin.Parse()

    // post build and repo data to webhook urls
}
```

Once we've parsed the build and repository details, and the webhook urls, we are ready to make the http requests. First, we'll need to construct our payload and marshal to JSON:

```go
data := struct{
    Repo  plugin.Repo  `json:"repo"`
    Build plugin.Build `json:"build"`
}{repo, build}

payload, _ := json.Marshal(&data)
```

And finally, for each URL in the list, we post the JSON payload:

```go
for _, url := range vargs.Urls {
     resp, _ := http.Post(url, "application/json", bytes.NewBuffer(payload))
     resp.Body.Close()
}
```

## Plugin image

Since plugins are distributed as Docker images, we'll need to create a `Dockerfile` for our plugin:

```dockerfile
FROM gliderlabs/alpine:3.1
RUN apk-install ca-certificates
ADD drone-webhook /bin/
ENTRYPOINT ["/bin/drone-webhook"]
```

We recommend using the `gliderlabs/alpine` base image due to its compact size. Plugins are downloaded automatically during build execution, therefore, smaller images and faster download times provide a better overall user experience.

## Plugin testing

We can quickly test our plugin from the command line, with dummy data, to make sure everything works. Note that you can send the JSON string over `stdin` instead of `arg[1]` only when testing:

```bash
go run main.go <<EOF
{
    "repo": {
        "owner": "octocat",
        "name": "hello-world",
        "full_name": "octocat/hello-world"
    },
    "build": {
        "number": 1,
        "status": "success"
    },
    "vargs": {
        "urls": [ "http://foo.com/post/to/this/url" ]
    }
}
EOF
```

## Complete example

Here is the complete Go program:

```go
import (
    "bytes"
    "encoding/json"
    "net/http"

    "github.com/drone/drone-plugin-go"
)

func main() {
    var repo = plugin.Repo{}
    var build = plugin.Build{}
    var vargs = struct {
        Urls []string `json:"urls"`
    }{}

    plugin.Param("repo", &repo)
    plugin.Param("build", &build)
    plugin.Param("vargs", &vargs)
    plugin.Parse()

    // data structure
    data := struct{
        Repo  plugin.Repo  `json:"repo"`
        Build plugin.Build `json:"build"`
    }{repo, build}

    // json payload that will be posted
    payload, _ := json.Marshal(&data)

    // post payload to each url
    for _, url := range vargs.Urls {
         resp, _ := http.Post(url, "application/json", bytes.NewBuffer(payload))
         resp.Body.Close()
    }
}
```

And the Dockerfile:

```dockerfile
FROM gliderlabs/alpine:3.1
RUN apk-install ca-certificates
ADD drone-webhook /bin/
ENTRYPOINT ["/bin/drone-webhook"]
```
