package main

import (
	"fmt"

	"github.com/anandvarma/namegen"
	"github.com/swarmlab-dev/go-tss/tssparty"
	"github.com/urfave/cli"
)

func keygenCmd() cli.Command {
	return cli.Command{
		Name:    "keygen",
		Aliases: []string{"k"},
		Usage:   "Keygen threshold ceremony to create a new party",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "bus",
				Value: "127.0.0.1:8080",
				Usage: "party bus URL",
			},
			cli.StringFlag{
				Name:  "s",
				Value: namegen.New().Get(),
				Usage: "keygen party session id",
			},
			cli.StringFlag{
				Name:  "p",
				Usage: "this peer id",
				Value: namegen.New().Get(),
			},
			cli.BoolFlag{
				Name:  "eddsa",
				Usage: "set keygen for eddsa (default is ecdsa)",
			},
			cli.IntFlag{
				Name:  "n",
				Value: 3,
				Usage: "number of shares",
			},
			cli.IntFlag{
				Name:  "t",
				Value: 2,
				Usage: "number of party necessary to sign (threshold)",
			},
		},
		Action: func(c *cli.Context) error {
			partyBusUrl := c.String("bus")
			sessionId := c.String("s")
			partyId := c.String("p")
			partycount := c.Int("n")
			threshold := c.Int("t")
			if threshold > partycount {
				fmt.Printf("threshold (t) must be lower than party count (n)")
			}

			if c.Bool("eddsa") {
				return tssparty.JoinEddsaKeygenParty(partyBusUrl, sessionId, partyId, partycount, threshold)
			} else {
				return tssparty.JoinEcdsaKeygenParty(partyBusUrl, sessionId, partyId, partycount, threshold)
			}
		},
	}
}
