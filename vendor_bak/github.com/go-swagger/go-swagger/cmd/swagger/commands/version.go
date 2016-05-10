package commands

import "fmt"

var (
	// Version for the swagger command
	Version string
)

// PrintVersion the command
type PrintVersion struct {
}

// Execute this command
func (p *PrintVersion) Execute(args []string) error {
	if Version == "" {
		fmt.Println("dev")
		return nil
	}
	fmt.Println(Version)
	return nil
}
