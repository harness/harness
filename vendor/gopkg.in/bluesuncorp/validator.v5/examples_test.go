package validator_test

import (
	"fmt"

	"../validator"
)

func ExampleValidate_new() {
	validator.New("validate", validator.BakedInValidators)
}

func ExampleValidate_addFunction() {
	// This should be stored somewhere globally
	var validate *validator.Validate

	validate = validator.New("validate", validator.BakedInValidators)

	fn := func(top interface{}, current interface{}, field interface{}, param string) bool {
		return field.(string) == "hello"
	}

	validate.AddFunction("valueishello", fn)

	message := "hello"
	err := validate.Field(message, "valueishello")
	fmt.Println(err)
	//Output:
	//<nil>
}

func ExampleValidate_field() {
	// This should be stored somewhere globally
	var validate *validator.Validate

	validate = validator.New("validate", validator.BakedInValidators)

	i := 0
	err := validate.Field(i, "gt=1,lte=10")
	fmt.Println(err.Field)
	fmt.Println(err.Tag)
	fmt.Println(err.Kind) // NOTE: Kind and Type can be different i.e. time Kind=struct and Type=time.Time
	fmt.Println(err.Type)
	fmt.Println(err.Param)
	fmt.Println(err.Value)
	//Output:
	//
	//gt
	//int
	//int
	//1
	//0
}

func ExampleValidate_struct() {
	// This should be stored somewhere globally
	var validate *validator.Validate

	validate = validator.New("validate", validator.BakedInValidators)

	type ContactInformation struct {
		Phone  string `validate:"required"`
		Street string `validate:"required"`
		City   string `validate:"required"`
	}

	type User struct {
		Name               string `validate:"required,excludesall=!@#$%^&*()_+-=:;?/0x2C"` // 0x2C = comma (,)
		Age                int8   `validate:"required,gt=0,lt=150"`
		Email              string `validate:"email"`
		ContactInformation []*ContactInformation
	}

	contactInfo := &ContactInformation{
		Street: "26 Here Blvd.",
		City:   "Paradeso",
	}

	user := &User{
		Name:               "Joey Bloggs",
		Age:                31,
		Email:              "joeybloggs@gmail.com",
		ContactInformation: []*ContactInformation{contactInfo},
	}

	structError := validate.Struct(user)
	for _, fieldError := range structError.Errors {
		fmt.Println(fieldError.Field) // Phone
		fmt.Println(fieldError.Tag)   // required
		//... and so forth
		//Output:
		//Phone
		//required
	}
}
