package main

import (
	"github.com/urfave/cli"
)

func resharingCmd() cli.Command {
	return cli.Command{
		Name:    "resharing",
		Aliases: []string{"r"},
		Usage:   "Resharing threshold ceremony to create fresh shares",
		Action: func(c *cli.Context) error {
			return nil
		},
	}
}
