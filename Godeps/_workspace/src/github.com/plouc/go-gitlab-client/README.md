go-gitlab-client
================

go-gitlab-client is a simple client written in golang to consume gitlab API.

[![Build Status](https://travis-ci.org/plouc/go-gitlab-client.png?branch=master)](https://travis-ci.org/plouc/go-gitlab-client)


##features

*	
	###Projects [gitlab api doc](http://api.gitlab.org/projects.html)
	* list projects
	* get single project

*	
	###Repositories [gitlab api doc](http://api.gitlab.org/repositories.html)
	* list repository branches
	* get single repository branch
	* list project repository tags
	* list repository commits
	* list project hooks
	* add/get/edit/rm project hook

*	
	###Users [gitlab api doc](http://api.gitlab.org/users.html)
	* get single user
	* manage user keys

*	
	###Deploy Keys [gitlab api doc](http://api.gitlab.org/deploy_keys.html)
	* list project deploy keys
	* add/get/rm project deploy key




##Installation

To install go-gitlab-client, use `go get`:

    go get github.com/plouc/go-gitlab-client

Import the `go-gitlab-client` package into your code:

```go
package whatever

import (
    "github.com/plouc/go-gitlab-client"
)
```


##Update

To update `go-gitlab-client`, use `go get -u`:

    go get -u github.com/plouc/go-gitlab-client


##Documentation

Visit the docs at http://godoc.org/github.com/plouc/go-gitlab-client


## Examples

You can play with the examples located in the `examples` directory

* [projects](https://github.com/plouc/go-gitlab-client/tree/master/examples/projects)
* [repositories](https://github.com/plouc/go-gitlab-client/tree/master/examples/repositories)