package main

import (
	"github.com/urfave/cli"
)

func signingCmd() cli.Command {
	return cli.Command{
		Name:    "signing",
		Aliases: []string{"s"},
		Usage:   "Signing threshold ceremony to sign a message",
		Action: func(c *cli.Context) error {
			return nil
		},
	}
}
