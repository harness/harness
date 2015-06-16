Package validator
================
[![Build Status](https://travis-ci.org/bluesuncorp/validator.svg?branch=v5.1)](https://travis-ci.org/bluesuncorp/validator)
[![GoDoc](https://godoc.org/gopkg.in/bluesuncorp/validator.v5?status.svg)](https://godoc.org/gopkg.in/bluesuncorp/validator.v5)

Package validator implements value validations for structs and individual fields based on tags.
It is also capable of Cross Field and Cross Struct validations.

Installation
============

Use go get.

	go get gopkg.in/bluesuncorp/validator.v5

or to update

	go get -u gopkg.in/bluesuncorp/validator.v5

Then import the validator package into your own code.

	import "gopkg.in/bluesuncorp/validator.v5"

Usage and documentation
=======================

Please see http://godoc.org/gopkg.in/bluesuncorp/validator.v5 for detailed usage docs.

How to Contribute
=================

There will always be a development branch for each version i.e. `v1-development`. In order to contribute, 
please make your pull requests against those branches.

If the changes being proposed or requested are breaking changes, please create an issue, for discussion 
or create a pull request against the highest development branch for example this package has a 
v1 and v1-development branch however, there will also be a v2-development brach even though v2 doesn't exist yet.

I strongly encourage everyone whom creates a custom validation function to contribute them and
help make this package even better.

License
=======
Distributed under MIT License, please see license file in code for more details.
