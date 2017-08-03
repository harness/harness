package ga

import "fmt"

func bool2str(val bool) string {
	if val {
		return "1"
	} else {
		return "0"
	}
}

func int2str(val int64) string {
	return fmt.Sprintf("%d", val)
}

func float2str(val float64) string {
	return fmt.Sprintf("%.6f", val)
}
