package assertions

import (
	"fmt"
	"reflect"
)

// ShouldHaveSameTypeAs receives exactly two parameters and compares their underlying types for equality.
func ShouldHaveSameTypeAs(actual interface{}, expected ...interface{}) string {
	if fail := need(1, expected); fail != success {
		return fail
	}

	first := reflect.TypeOf(actual)
	second := reflect.TypeOf(expected[0])

	if equal := ShouldEqual(first, second); equal != success {
		return serializer.serialize(second, first, fmt.Sprintf(shouldHaveBeenA, actual, second, first))
	}
	return success
}

// ShouldNotHaveSameTypeAs receives exactly two parameters and compares their underlying types for inequality.
func ShouldNotHaveSameTypeAs(actual interface{}, expected ...interface{}) string {
	if fail := need(1, expected); fail != success {
		return fail
	}

	first := reflect.TypeOf(actual)
	second := reflect.TypeOf(expected[0])

	if equal := ShouldEqual(first, second); equal == success {
		return fmt.Sprintf(shouldNotHaveBeenA, actual, second)
	}
	return success
}

// ShouldImplement receives exactly two parameters and ensures
// that the first implements the interface type of the second.
func ShouldImplement(actual interface{}, expectedList ...interface{}) string {
	if fail := need(1, expectedList); fail != success {
		return fail
	}
	expected := expectedList[0]
	if fail := ShouldBeNil(expected); fail != success {
		return shouldCompareWithInterfacePointer
	}
	expectedType := reflect.TypeOf(expected)
	if fail := ShouldNotBeNil(expectedType); fail != success {
		return shouldCompareWithInterfacePointer
	}

	expectedInterface := expectedType.Elem()
	actualType := reflect.TypeOf(actual)

	if actualType == nil {
		return fmt.Sprintf(shouldHaveImplemented, expectedInterface, actual)
	}
	if fail := ShouldEqual(actualType.Kind(), reflect.Ptr); fail != success {
		return fmt.Sprintf(shouldHaveImplemented, expectedInterface, actual)
	}
	if !actualType.Implements(expectedInterface) {
		return fmt.Sprintf(shouldHaveImplemented, expectedInterface, actualType)
	}
	return success
}

// ShouldNotImplement receives exactly two parameters and ensures
// that the first does NOT implement the interface type of the second.
func ShouldNotImplement(actual interface{}, expectedList ...interface{}) string {
	if fail := need(1, expectedList); fail != success {
		return fail
	}
	expected := expectedList[0]
	if fail := ShouldBeNil(expected); fail != success {
		return shouldCompareWithInterfacePointer
	}
	expectedType := reflect.TypeOf(expected)
	if fail := ShouldNotBeNil(expectedType); fail != success {
		return shouldCompareWithInterfacePointer
	}

	expectedInterface := expectedType.Elem()
	actualType := reflect.TypeOf(actual)

	if actualType == nil {
		return success
	}
	if actualType.Implements(expectedInterface) {
		return fmt.Sprintf(shouldNotHaveImplemented, actualType, expectedInterface)
	}
	return success
}
