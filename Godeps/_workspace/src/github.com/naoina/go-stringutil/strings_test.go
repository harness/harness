package stringutil_test

import (
	"reflect"
	"testing"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/naoina/go-stringutil"
)

func TestToUpperCamelCase(t *testing.T) {
	for _, v := range []struct {
		input, expect string
	}{
		{"", ""},
		{"thequickbrownfoxoverthelazydog", "Thequickbrownfoxoverthelazydog"},
		{"thequickbrownfoxoverthelazydoG", "ThequickbrownfoxoverthelazydoG"},
		{"thequickbrownfoxoverthelazydo_g", "ThequickbrownfoxoverthelazydoG"},
		{"TheQuickBrownFoxJumpsOverTheLazyDog", "TheQuickBrownFoxJumpsOverTheLazyDog"},
		{"the_quick_brown_fox_jumps_over_the_lazy_dog", "TheQuickBrownFoxJumpsOverTheLazyDog"},
		{"the_Quick_Brown_Fox_Jumps_Over_The_Lazy_Dog", "TheQuickBrownFoxJumpsOverTheLazyDog"},
		{"ｔｈｅ_ｑｕｉｃｋ_ｂｒｏｗｎ_ｆｏｘ_ｏｖｅｒ_ｔｈｅ_ｌａｚｙ_ｄｏｇ", "ＴｈｅＱｕｉｃｋＢｒｏｗｎＦｏｘＯｖｅｒＴｈｅＬａｚｙＤｏｇ"},
	} {
		actual := stringutil.ToUpperCamelCase(v.input)
		expect := v.expect
		if !reflect.DeepEqual(actual, expect) {
			t.Errorf(`stringutil.ToUpperCamelCase(%#v) => %#v; want %#v`, v.input, actual, expect)
		}
	}
}

func TestToUpperCamelCaseASCII(t *testing.T) {
	for _, v := range []struct {
		input, expect string
	}{
		{"", ""},
		{"thequickbrownfoxoverthelazydog", "Thequickbrownfoxoverthelazydog"},
		{"thequickbrownfoxoverthelazydoG", "ThequickbrownfoxoverthelazydoG"},
		{"thequickbrownfoxoverthelazydo_g", "ThequickbrownfoxoverthelazydoG"},
		{"TheQuickBrownFoxJumpsOverTheLazyDog", "TheQuickBrownFoxJumpsOverTheLazyDog"},
		{"the_quick_brown_fox_jumps_over_the_lazy_dog", "TheQuickBrownFoxJumpsOverTheLazyDog"},
		{"the_Quick_Brown_Fox_Jumps_Over_The_Lazy_Dog", "TheQuickBrownFoxJumpsOverTheLazyDog"},
	} {
		actual := stringutil.ToUpperCamelCaseASCII(v.input)
		expect := v.expect
		if !reflect.DeepEqual(actual, expect) {
			t.Errorf(`stringutil.ToUpperCamelCaseASCII(%#v) => %#v; want %#v`, v.input, actual, expect)
		}
	}
}

func TestToSnakeCase(t *testing.T) {
	for _, v := range []struct {
		input, expect string
	}{
		{"", ""},
		{"thequickbrownfoxjumpsoverthelazydog", "thequickbrownfoxjumpsoverthelazydog"},
		{"Thequickbrownfoxjumpsoverthelazydog", "thequickbrownfoxjumpsoverthelazydog"},
		{"ThequickbrownfoxjumpsoverthelazydoG", "thequickbrownfoxjumpsoverthelazydo_g"},
		{"TheQuickBrownFoxJumpsOverTheLazyDog", "the_quick_brown_fox_jumps_over_the_lazy_dog"},
		{"the_quick_brown_fox_jumps_over_the_lazy_dog", "the_quick_brown_fox_jumps_over_the_lazy_dog"},
		{"ＴｈｅＱｕｉｃｋＢｒｏｗｎＦｏｘＯｖｅｒＴｈｅＬａｚｙＤｏｇ", "ｔｈｅ_ｑｕｉｃｋ_ｂｒｏｗｎ_ｆｏｘ_ｏｖｅｒ_ｔｈｅ_ｌａｚｙ_ｄｏｇ"},
	} {
		actual := stringutil.ToSnakeCase(v.input)
		expect := v.expect
		if !reflect.DeepEqual(actual, expect) {
			t.Errorf(`stringutil.ToSnakeCase(%#v) => %#v; want %#v`, v.input, actual, expect)
		}
	}
}

func TestToSnakeCaseASCII(t *testing.T) {
	for _, v := range []struct {
		input, expect string
	}{
		{"", ""},
		{"thequickbrownfoxjumpsoverthelazydog", "thequickbrownfoxjumpsoverthelazydog"},
		{"Thequickbrownfoxjumpsoverthelazydog", "thequickbrownfoxjumpsoverthelazydog"},
		{"ThequickbrownfoxjumpsoverthelazydoG", "thequickbrownfoxjumpsoverthelazydo_g"},
		{"TheQuickBrownFoxJumpsOverTheLazyDog", "the_quick_brown_fox_jumps_over_the_lazy_dog"},
		{"the_quick_brown_fox_jumps_over_the_lazy_dog", "the_quick_brown_fox_jumps_over_the_lazy_dog"},
	} {
		actual := stringutil.ToSnakeCaseASCII(v.input)
		expect := v.expect
		if !reflect.DeepEqual(actual, expect) {
			t.Errorf(`stringutil.ToSnakeCaseASCII(%#v) => %#v; want %#v`, v.input, actual, expect)
		}
	}
}
