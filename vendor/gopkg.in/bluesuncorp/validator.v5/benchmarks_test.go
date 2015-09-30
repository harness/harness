package validator

import "testing"

func BenchmarkValidateField(b *testing.B) {
	for n := 0; n < b.N; n++ {
		validate.Field("1", "len=1")
	}
}

func BenchmarkValidateStructSimple(b *testing.B) {

	type Foo struct {
		StringValue string `validate:"min=5,max=10"`
		IntValue    int    `validate:"min=5,max=10"`
	}

	validFoo := &Foo{StringValue: "Foobar", IntValue: 7}
	invalidFoo := &Foo{StringValue: "Fo", IntValue: 3}

	for n := 0; n < b.N; n++ {
		validate.Struct(validFoo)
		validate.Struct(invalidFoo)
	}
}

func BenchmarkTemplateParallelSimple(b *testing.B) {

	type Foo struct {
		StringValue string `validate:"min=5,max=10"`
		IntValue    int    `validate:"min=5,max=10"`
	}

	validFoo := &Foo{StringValue: "Foobar", IntValue: 7}
	invalidFoo := &Foo{StringValue: "Fo", IntValue: 3}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			validate.Struct(validFoo)
			validate.Struct(invalidFoo)
		}
	})
}

func BenchmarkValidateStructLarge(b *testing.B) {

	tFail := &TestString{
		Required:  "",
		Len:       "",
		Min:       "",
		Max:       "12345678901",
		MinMax:    "",
		Lt:        "0123456789",
		Lte:       "01234567890",
		Gt:        "1",
		Gte:       "1",
		OmitEmpty: "12345678901",
		Sub: &SubTest{
			Test: "",
		},
		Anonymous: struct {
			A string `validate:"required"`
		}{
			A: "",
		},
		Iface: &Impl{
			F: "12",
		},
	}

	tSuccess := &TestString{
		Required:  "Required",
		Len:       "length==10",
		Min:       "min=1",
		Max:       "1234567890",
		MinMax:    "12345",
		Lt:        "012345678",
		Lte:       "0123456789",
		Gt:        "01234567890",
		Gte:       "0123456789",
		OmitEmpty: "",
		Sub: &SubTest{
			Test: "1",
		},
		SubIgnore: &SubTest{
			Test: "",
		},
		Anonymous: struct {
			A string `validate:"required"`
		}{
			A: "1",
		},
		Iface: &Impl{
			F: "123",
		},
	}

	for n := 0; n < b.N; n++ {
		validate.Struct(tSuccess)
		validate.Struct(tFail)
	}
}

func BenchmarkTemplateParallelLarge(b *testing.B) {

	tFail := &TestString{
		Required:  "",
		Len:       "",
		Min:       "",
		Max:       "12345678901",
		MinMax:    "",
		Lt:        "0123456789",
		Lte:       "01234567890",
		Gt:        "1",
		Gte:       "1",
		OmitEmpty: "12345678901",
		Sub: &SubTest{
			Test: "",
		},
		Anonymous: struct {
			A string `validate:"required"`
		}{
			A: "",
		},
		Iface: &Impl{
			F: "12",
		},
	}

	tSuccess := &TestString{
		Required:  "Required",
		Len:       "length==10",
		Min:       "min=1",
		Max:       "1234567890",
		MinMax:    "12345",
		Lt:        "012345678",
		Lte:       "0123456789",
		Gt:        "01234567890",
		Gte:       "0123456789",
		OmitEmpty: "",
		Sub: &SubTest{
			Test: "1",
		},
		SubIgnore: &SubTest{
			Test: "",
		},
		Anonymous: struct {
			A string `validate:"required"`
		}{
			A: "1",
		},
		Iface: &Impl{
			F: "123",
		},
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			validate.Struct(tSuccess)
			validate.Struct(tFail)
		}
	})
}
