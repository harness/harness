# Pre-Requisites

Install the latest stable version of Node and Go version 1.17 or higher, and then install the below Go programs. Ensure the GOPATH [bin directory](https://go.dev/doc/gopath_code#GOPATH) is added to your PATH.

```text
$ go install github.com/golang/mock/mockgen@latest
$ go install github.com/google/wire/cmd/wire@latest
```

# Build

Build the user interface:

```text
$ pushd web
$ npm install
$ npm run build
$ popd
```

Build the server and command line tools:

```text
$ go generate ./...
$ go build -o release/my-app
```

# Test

Execute the unit tests:

```text
$ go generate ./...
$ go test -v -cover ./...
```

# Run

This project supports all operating systems and architectures supported by Go.  This means you can build and run the system on your machine; docker containers are not required for local development and testing.

Start the server at `localhost:3000`

```text
$ release/my-app server
```

# User Interface

This project includes a simple user interface for interacting with the system. When you run the application, you can access the user interface by navigating to `http://localhost:3000` in your browser.

# Swagger

This project includes a swagger specification. When you run the application, you can access the swagger specification by navigating to `http://localhost:3000/swagger` in your browser.

# Command Line

This project includes simple command line tools for interacting with the system. Please remember that you must start the server before you can execute commands.

Register a new user:

```text
$ release/my-app register
```

Login to the application:

```text
$ release/my-app login
```

Logout from the application:

```text
$ release/my-app logout
```

View your account details:

```text
$ release/my-app account
```

Generate a peronsal access token:

```text
$ release/my-app token
```

Create a pipeline:

```text
$ release/my-app pipeline create <name>
```

List pipelines:

```text
$ release/my-app pipeline ls
```

Debug and output http responses from the server:

```text
$ DEBUG=true release/my-app pipeline ls
```

View all commands:

```text
$ release/my-app --help
```
