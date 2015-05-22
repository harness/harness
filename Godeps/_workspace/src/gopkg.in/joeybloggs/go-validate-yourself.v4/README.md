Package go-validate-yourself
================
[![Build Status](https://travis-ci.org/bluesuncorp/go-validate-yourself.svg?branch=v4)](https://travis-ci.org/bluesuncorp/go-validate-yourself)
[![GoDoc](https://godoc.org/gopkg.in/bluesuncorp/go-validate-yourself.v4?status.svg)](https://godoc.org/gopkg.in/bluesuncorp/go-validate-yourself.v4)

Package validator implements value validations for structs and individual fields based on tags.

Installation
============

Just use go get.

	go get gopkg.in/bluesuncorp/go-validate-yourself.v4

or to update

	go get -u gopkg.in/bluesuncorp/go-validate-yourself.v4

And then just import the package into your own code.

	import "gopkg.in/bluesuncorp/go-validate-yourself.v4"

Usage
=====

Please see http://godoc.org/gopkg.in/bluesuncorp/go-validate-yourself.v4 for detailed usage docs.

Contributing
============

There will be a development branch for each version of this package i.e. v1-development, please
make your pull requests against those branches.

If changes are breaking please create an issue, for discussion and create a pull request against
the highest development branch for example this package has a v1 and v1-development branch
however, there will also be a v2-development brach even though v2 doesn't exist yet.

I strongly encourage everyone whom creates a custom validation function to contribute them and
help make this package even better.

License
=======
Distributed under MIT License, please see license file in code for more details.