// +build windows plan9 solaris

package flags

func getTerminalColumns() int {
	return 80
}
