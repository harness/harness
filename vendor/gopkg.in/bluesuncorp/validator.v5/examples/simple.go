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
