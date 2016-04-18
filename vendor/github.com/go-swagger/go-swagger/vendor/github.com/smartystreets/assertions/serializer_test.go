package assertions

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestSerializerCreatesSerializedVersionOfAssertionResult(t *testing.T) {
	thing1 := Thing1{"Hi"}
	thing2 := Thing2{"Bye"}
	message := "Super-hip failure message."
	serializer := newSerializer()

	actualResult := serializer.serialize(thing1, thing2, message)

	expectedResult, _ := json.Marshal(FailureView{
		Message:  message,
		Expected: fmt.Sprintf("%+v", thing1),
		Actual:   fmt.Sprintf("%+v", thing2),
	})

	if actualResult != string(expectedResult) {
		t.Errorf("\nExpected: %s\nActual:   %s", string(expectedResult), actualResult)
	}

	actualResult = serializer.serializeDetailed(thing1, thing2, message)
	expectedResult, _ = json.Marshal(FailureView{
		Message:  message,
		Expected: fmt.Sprintf("%#v", thing1),
		Actual:   fmt.Sprintf("%#v", thing2),
	})
	if actualResult != string(expectedResult) {
		t.Errorf("\nExpected: %s\nActual:   %s", string(expectedResult), actualResult)
	}
}
