package main

import (
	"fmt"
	"os"

	"github.com/andrewpillar/cli"
)

func mainCommand(c cli.Command) {

}

func main() {
	c := cli.New()

	cmd := c.Main(mainCommand)

	cmd.AddFlag(&cli.Flag{
		Name:     "config",
		Short:    "-c",
		Long:     "--config",
		Argument: true,
		Default:  "thrall-server.yml",
	})

	if err := c.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
