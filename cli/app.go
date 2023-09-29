package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "tss-cli"
	app.Usage = "A simple command line to play with mpc-tss"
	app.Version = "1.0.0"

	app.Commands = []cli.Command{
		keygenCmd(),
		signingCmd(),
		resharingCmd(),
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}
