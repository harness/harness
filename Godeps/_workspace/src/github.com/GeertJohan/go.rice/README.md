## go.rice

go.rice is a [Go](http://golang.org) package that makes working with resources such as html,js,css,images and templates very easy. During development `go.rice` will load required files directly from disk. Upon deployment it is easy to add all resource files to a executable using the `rice` tool, without changing the source code for your package.

### What does it do?
The first thing go.rice does is finding the correct absolute path for your resource files. Say you are executing go binary in your home directory, but your html files are located in `$GOPATH/src/webApplication/html-files`. `go.rice` will lookup the aboslute path for that directory. The only thing you have to do is include the resources using `rice.FindBox("html-files")`.

This only works when the source is available to the machine executing the binary. This is always the case when the binary was installed with `go get` or `go install`. It might happen that you wish to simply provide a binary, without source. The `rice` tool analyses source code and finds call's to `rice.FindBox(..)` and adds the required directories to the executable binary. There are several ways to add these resources. You can 'embed' by generating go source code, or append the resource to the executable.

### Installation

Use `go get` for the package and `go install` for the tool.
```
go get github.com/GeertJohan/go.rice
go get github.com/GeertJohan/go.rice/rice
```

### Package usage

Import the package: `import "github.com/GeertJohan/go.rice"`

**Serving a static content folder over HTTP with a rice Box**
```go
http.Handle("/", http.FileServer(rice.MustFindBox("http-files").HTTPBox()))
http.ListenAndServe(":8080", nil)
```

**Loading a template**
```go
// find/create a rice.Box
templateBox, err := rice.FindBox("example-templates")
if err != nil {
	log.Fatal(err)
}
// get file contents as string
templateString, err := templateBox.String("message.tmpl")
if err != nil {
	log.Fatal(err)
}
// parse and execute the template
tmplMessage, err := template.New("message").Parse(templateString)
if err != nil {
	log.Fatal(err)
}
tmplMessage.Execute(os.Stdout, map[string]string{"Message": "Hello, world!"})

```

### Tool usage
The `rice` tool lets you add the resources to a binary executable so the files are not loaded from the filesystem anymore. This creates a 'standalone' executable. There's several ways to add the resources to a binary, each has pro's and con's but all will work without changing your source code. `go.rice` will figure it all out for you.

**Embed resources in Go source**

This option is pre-build, it generates Go source files that are compiled into the binary.

Run `rice embed` to generate Go source that contains all required resources. Afterwards run `go build` to create a standalone executable.

**Append resources to executable**

_Does not work on windows (yet)_

This options is post-build, it appends the resources to the binary. It makes compilation a lot faster and can be used with large resource files.

Appending requires `zip` to be installed.

Run the following commands to create a standalone executable.
```
go build -o example
rice append --exec example
```

#### Help information
Run `rice -h` for information about all options.
You can run the -h option for each sub-command, e.g. `rice append -h`.

### Order of preference
When opening a new box, the rice pkg tries to locate it using the following order:

 - embedded in generated go source
 - embedded with .a object files (not available yet)
 - appended as zip
 - 'live' from filesystem


### Licence
This project is licensed under a Simplified BSD license. Please read the [LICENSE file][license].


### TODO & Development
This package is not completed yet. Though it already provides working embedding, some important featuers are still missing.
 - implement Readdir() correctly on virtualDir
 - implement Seek() for zipFile
 - implement embedding with .a object files
 - automated testing with TravisCI or Drone **important**
 - in-code TODO's
 - find boxes in imported packages

Less important stuff:
 - rice.FindSingle(..) that loads and embeds a single file as oposed to a complete directory. It should have methods .String(), .Bytes() and .File()
 - The rice tool uses a simple regexp to find calls to `rice.FindBox(..)`, this should be changed to `go/ast` or maybe `go.tools/oracle`?
 - idea, os/arch dependent embeds. rice checks if embedding file has _os_arch or build flags. If box is not requested by file without buildflags, then the buildflags are applied to the embed file.
 - store meta information for appended (zip) files (mod time, etc)

### Package documentation

You will find package documentation at [godoc.org/github.com/GeertJohan/go.rice][godoc].


 [license]: https://github.com/GeertJohan/go.rice/blob/master/LICENSE
 [godoc]: http://godoc.org/github.com/GeertJohan/go.rice
 
