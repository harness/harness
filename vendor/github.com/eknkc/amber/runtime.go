package amber

import (
	"encoding/json"
	"fmt"
	"html/template"
	"reflect"
)

var FuncMap = template.FuncMap{
	"__amber_add":   runtime_add,
	"__amber_sub":   runtime_sub,
	"__amber_mul":   runtime_mul,
	"__amber_quo":   runtime_quo,
	"__amber_rem":   runtime_rem,
	"__amber_minus": runtime_minus,
	"__amber_plus":  runtime_plus,
	"__amber_eql":   runtime_eql,
	"__amber_gtr":   runtime_gtr,
	"__amber_lss":   runtime_lss,

	"json":      runtime_json,
	"unescaped": runtime_unescaped,
}

func runtime_add(x, y interface{}) interface{} {
	vx, vy := reflect.ValueOf(x), reflect.ValueOf(y)
	switch vx.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.Int() + vy.Int()
			case reflect.Float32, reflect.Float64:
				return float64(vx.Int()) + vy.Float()
			case reflect.String:
				return fmt.Sprintf("%d%s", vx.Int(), vy.String())
			}
		}
	case reflect.Float32, reflect.Float64:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.Float() + float64(vy.Int())
			case reflect.Float32, reflect.Float64:
				return vx.Float() + vy.Float()
			case reflect.String:
				return fmt.Sprintf("%f%s", vx.Float(), vy.String())
			}
		}
	case reflect.String:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return fmt.Sprintf("%s%d", vx.String(), vy.Int())
			case reflect.Float32, reflect.Float64:
				return fmt.Sprintf("%s%f", vx.String(), vy.Float())
			case reflect.String:
				return fmt.Sprintf("%s%s", vx.String(), vy.String())
			}
		}
	}

	return "<nil>"
}

func runtime_sub(x, y interface{}) interface{} {
	vx, vy := reflect.ValueOf(x), reflect.ValueOf(y)
	switch vx.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.Int() - vy.Int()
			case reflect.Float32, reflect.Float64:
				return float64(vx.Int()) - vy.Float()
			}
		}
	case reflect.Float32, reflect.Float64:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.Float() - float64(vy.Int())
			case reflect.Float32, reflect.Float64:
				return vx.Float() - vy.Float()
			}
		}
	}

	return "<nil>"
}

func runtime_mul(x, y interface{}) interface{} {
	vx, vy := reflect.ValueOf(x), reflect.ValueOf(y)
	switch vx.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.Int() * vy.Int()
			case reflect.Float32, reflect.Float64:
				return float64(vx.Int()) * vy.Float()
			}
		}
	case reflect.Float32, reflect.Float64:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.Float() * float64(vy.Int())
			case reflect.Float32, reflect.Float64:
				return vx.Float() * vy.Float()
			}
		}
	}

	return "<nil>"
}

func runtime_quo(x, y interface{}) interface{} {
	vx, vy := reflect.ValueOf(x), reflect.ValueOf(y)
	switch vx.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.Int() / vy.Int()
			case reflect.Float32, reflect.Float64:
				return float64(vx.Int()) / vy.Float()
			}
		}
	case reflect.Float32, reflect.Float64:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.Float() / float64(vy.Int())
			case reflect.Float32, reflect.Float64:
				return vx.Float() / vy.Float()
			}
		}
	}

	return "<nil>"
}

func runtime_rem(x, y interface{}) interface{} {
	vx, vy := reflect.ValueOf(x), reflect.ValueOf(y)
	switch vx.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.Int() % vy.Int()
			}
		}
	}

	return "<nil>"
}

func runtime_minus(x interface{}) interface{} {
	vx := reflect.ValueOf(x)
	switch vx.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
		return -vx.Int()
	case reflect.Float32, reflect.Float64:
		return -vx.Float()
	}

	return "<nil>"
}

func runtime_plus(x interface{}) interface{} {
	vx := reflect.ValueOf(x)
	switch vx.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
		return +vx.Int()
	case reflect.Float32, reflect.Float64:
		return +vx.Float()
	}

	return "<nil>"
}

func runtime_eql(x, y interface{}) bool {
	vx, vy := reflect.ValueOf(x), reflect.ValueOf(y)
	switch vx.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.Int() == vy.Int()
			case reflect.Float32, reflect.Float64:
				return float64(vx.Int()) == vy.Float()
			case reflect.String:
				return fmt.Sprintf("%d", vx.Int()) == vy.String()
			}
		}
	case reflect.Float32, reflect.Float64:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.Float() == float64(vy.Int())
			case reflect.Float32, reflect.Float64:
				return vx.Float() == vy.Float()
			case reflect.String:
				return fmt.Sprintf("%f", vx.Float()) == vy.String()
			}
		}
	case reflect.String:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.String() == fmt.Sprintf("%d", vy.Int())
			case reflect.Float32, reflect.Float64:
				return vx.String() == fmt.Sprintf("%f", vy.Float())
			case reflect.String:
				return vx.String() == fmt.Sprintf("%s", vy.String())
			}
		}
	case reflect.Bool:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.Bool() && vy.Int() != 0
			case reflect.Bool:
				return vx.Bool() == vy.Bool()
			}
		}
	}

	return false
}

func runtime_lss(x, y interface{}) bool {
	vx, vy := reflect.ValueOf(x), reflect.ValueOf(y)
	switch vx.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.Int() < vy.Int()
			case reflect.Float32, reflect.Float64:
				return float64(vx.Int()) < vy.Float()
			case reflect.String:
				return fmt.Sprintf("%d", vx.Int()) < vy.String()
			}
		}
	case reflect.Float32, reflect.Float64:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.Float() < float64(vy.Int())
			case reflect.Float32, reflect.Float64:
				return vx.Float() < vy.Float()
			case reflect.String:
				return fmt.Sprintf("%f", vx.Float()) < vy.String()
			}
		}
	case reflect.String:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.String() < fmt.Sprintf("%d", vy.Int())
			case reflect.Float32, reflect.Float64:
				return vx.String() < fmt.Sprintf("%f", vy.Float())
			case reflect.String:
				return vx.String() < vy.String()
			}
		}
	}

	return false
}

func runtime_gtr(x, y interface{}) bool {
	return !runtime_lss(x, y) && !runtime_eql(x, y)
}

func runtime_json(x interface{}) (res string, err error) {
	bres, err := json.Marshal(x)
	res = string(bres)
	return
}

func runtime_unescaped(x string) interface{} {
	return template.HTML(x)
}
