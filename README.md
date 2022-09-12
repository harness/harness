# Pre-Requisites

Install the latest stable version of Node and Go version 1.17 or higher, and then install the below Go programs. Ensure the GOPATH [bin directory](https://go.dev/doc/gopath_code#GOPATH) is added to your PATH.

```bash
$ make all
```

# Build

Build the user interface:

```bash
$ pushd web
$ yarn install
$ yarn run build
$ popd
```

Build the server and command line tools:

```bash
$ go generate ./...
$ go build -o release/gitness
```

# Test

Execute the unit tests:

```bash
$ make test
```

# Run

This project supports all operating systems and architectures supported by Go.  This means you can build and run the system on your machine; docker containers are not required for local development and testing.

Start the server at `localhost:3000`

```bash
$ release/gitness server
```

# User Interface

This project includes a simple user interface for interacting with the system. When you run the application, you can access the user interface by navigating to `http://localhost:3000` in your browser.

# Swagger

This project includes a swagger specification. When you run the application, you can access the swagger specification by navigating to `http://localhost:3000/swagger` in your browser.

# Command Line

This project includes simple command line tools for interacting with the system. Please remember that you must start the server before you can execute commands.

Register a new user:

```bash
$ release/gitness register
```

Login to the application:

```bash
$ release/gitness login
```

Logout from the application:

```bash
$ release/gitness logout
```

View your account details:

```bash
$ release/gitness account
```

Generate a personal access token:

```bash
$ release/gitness token
```

Create a pipeline:

```bash
$ release/gitness pipeline create <name>
```

List pipelines:

```bash
$ release/gitness pipeline ls
```

Debug and output http responses from the server:

```bash
$ DEBUG=true release/gitness pipeline ls
```

View all commands:

```bash
$ release/gitness --help
```
