Package validator
================

[![Join the chat at https://gitter.im/bluesuncorp/validator](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/bluesuncorp/validator?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![Build Status](https://semaphoreci.com/api/v1/projects/ec20115f-ef1b-4c7d-9393-cc76aba74eb4/487382/badge.svg)](https://semaphoreci.com/joeybloggs/validator)
[![Coverage Status](https://coveralls.io/repos/bluesuncorp/validator/badge.svg?branch=v5)](https://coveralls.io/r/bluesuncorp/validator?branch=v5)
[![GoDoc](https://godoc.org/gopkg.in/bluesuncorp/validator.v5?status.svg)](https://godoc.org/gopkg.in/bluesuncorp/validator.v5)

Package validator implements value validations for structs and individual fields based on tags.

It has the following **unique** features:

-   Cross Field and Cross Struct validations.  
-   Slice, Array and Map diving, which allows any or all levels of a multidimensional field to be validated.  
-   Handles type interface by determining it's underlying type prior to validation.  

Installation
------------

Use go get.

	go get gopkg.in/bluesuncorp/validator.v5

or to update

	go get -u gopkg.in/bluesuncorp/validator.v5

Then import the validator package into your own code.

	import "gopkg.in/bluesuncorp/validator.v5"

Usage and documentation
------

Please see http://godoc.org/gopkg.in/bluesuncorp/validator.v5 for detailed usage docs.

##### Example:
```go
package main

import (
	"fmt"

	"gopkg.in/bluesuncorp/validator.v5"
)

// User contains user information
type User struct {
	FirstName      string     `validate:"required"`
	LastName       string     `validate:"required"`
	Age            uint8      `validate:"gte=0,lte=130"`
	Email          string     `validate:"required,email"`
	FavouriteColor string     `validate:"hexcolor|rgb|rgba"`
	Addresses      []*Address `validate:"required,dive,required"` // a person can have a home and cottage...
}

// Address houses a users address information
type Address struct {
	Street string `validate:"required"`
	City   string `validate:"required"`
	Planet string `validate:"required"`
	Phone  string `validate:"required"`
}

var validate *validator.Validate

func main() {

	validate = validator.New("validate", validator.BakedInValidators)

	address := &Address{
		Street: "Eavesdown Docks",
		Planet: "Persphone",
		Phone:  "none",
	}

	user := &User{
		FirstName:      "Badger",
		LastName:       "Smith",
		Age:            135,
		Email:          "Badger.Smith@gmail.com",
		FavouriteColor: "#000",
		Addresses:      []*Address{address},
	}

	// returns nil or *StructErrors
	errs := validate.Struct(user)

	if errs != nil {

		// err will be of type *FieldError
		err := errs.Errors["Age"]
		fmt.Println(err.Error()) // output: Field validation for "Age" failed on the "lte" tag
		fmt.Println(err.Field)   // output: Age
		fmt.Println(err.Tag)     // output: lte
		fmt.Println(err.Kind)    // output: uint8
		fmt.Println(err.Type)    // output: uint8
		fmt.Println(err.Param)   // output: 130
		fmt.Println(err.Value)   // output: 135

		// or if you prefer you can use the Flatten function
		// NOTE: I find this usefull when using a more hard static approach of checking field errors.
		// The above, is best for passing to some generic code to say parse the errors. i.e. I pass errs
		// to a routine which loops through the errors, creates and translates the error message into the
		// users locale and returns a map of map[string]string // field and error which I then use
		// within the HTML rendering.

		flat := errs.Flatten()
		fmt.Println(flat) // output: map[Age:Field validation for "Age" failed on the "lte" tag Addresses[0].Address.City:Field validation for "City" failed on the "required" tag]
		err = flat["Addresses[0].Address.City"]
		fmt.Println(err.Field) // output: City
		fmt.Println(err.Tag)   // output: required
		fmt.Println(err.Kind)  // output: string
		fmt.Println(err.Type)  // output: string
		fmt.Println(err.Param) // output:
		fmt.Println(err.Value) // output:

		// from here you can create your own error messages in whatever language you wish
		return
	}

	// save user to database
}
```

Benchmarks
------
###### Run on MacBook Pro (Retina, 15-inch, Late 2013) 2.6 GHz Intel Core i7 16 GB 1600 MHz DDR3
```go
$ go test -cpu=4 -bench=. -benchmem=true
PASS
BenchmarkValidateField-4	 		 3000000	       429 ns/op	     192 B/op	       2 allocs/op
BenchmarkValidateStructSimple-4	  	  500000	      2877 ns/op	     657 B/op	      10 allocs/op
BenchmarkTemplateParallelSimple-4	  500000	      3097 ns/op	     657 B/op	      10 allocs/op
BenchmarkValidateStructLarge-4	  	  100000	     15228 ns/op	    4350 B/op	      62 allocs/op
BenchmarkTemplateParallelLarge-4	  100000	     14257 ns/op	    4354 B/op	      62 allocs/op
```

How to Contribute
------

There will always be a development branch for each version i.e. `v1-development`. In order to contribute, 
please make your pull requests against those branches.

If the changes being proposed or requested are breaking changes, please create an issue, for discussion 
or create a pull request against the highest development branch for example this package has a 
v1 and v1-development branch however, there will also be a v2-development brach even though v2 doesn't exist yet.

I strongly encourage everyone whom creates a custom validation function to contribute them and
help make this package even better.

License
------
Distributed under MIT License, please see license file in code for more details.
