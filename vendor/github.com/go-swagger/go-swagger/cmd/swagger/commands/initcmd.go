package commands

import "github.com/go-swagger/go-swagger/cmd/swagger/commands/initcmd"

// InitCmd is a command namespace for initializing things like a swagger spec.
type InitCmd struct {
	Model *initcmd.Spec `command:"spec"`
}

func (i *InitCmd) Execute(args []string) error {
	return nil
}
