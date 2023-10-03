package main

import (
	"fmt"
	"io"
	"os"

	"github.com/anandvarma/namegen"
	"github.com/swarmlab-dev/go-tss/tssparty"
	"github.com/urfave/cli"
)

func signingCmd() cli.Command {
	return cli.Command{
		Name:    "signing",
		Aliases: []string{"s"},
		Usage:   "Signing threshold ceremony to sign a message",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "bus",
				Value: "127.0.0.1:8080",
				Usage: "party bus URL",
			},
			cli.StringFlag{
				Name:  "s",
				Value: namegen.New().Get(),
				Usage: "signing party session id",
			},
			cli.StringFlag{
				Name:  "p",
				Usage: "this peer id",
				Value: namegen.New().Get(),
			},
			cli.StringFlag{
				Name:  "k",
				Usage: "this peer's key share",
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
			cli.StringFlag{
				Name:  "msg",
				Value: "",
				Usage: "message to sign with threshold algorithm",
			},
		},
		Action: func(c *cli.Context) error {
			message := c.String("msg")
			partyBusUrl := c.String("bus")
			sessionId := c.String("s")
			partyId := c.String("p")
			partycount := c.Int("n")
			threshold := c.Int("t")
			if threshold > partycount {
				return fmt.Errorf("threshold (t) must be lower than party count (n)")
			}

			keyShare := c.String("k")
			if keyShare == "-" {
				keyShareB, err := io.ReadAll(os.Stdin)
				if err != nil {
					return err
				}
				keyShare = string(keyShareB)
			}

			if c.Bool("eddsa") {
				local, err := tssparty.JoinEddsaSigningParty(partyBusUrl, sessionId, message, keyShare, partyId, partycount, threshold)
				if err != nil {
					return err
				}
				fmt.Printf("%s\n", local)
			} else {
				local, err := tssparty.JoinEcdsaSigningParty(partyBusUrl, sessionId, message, keyShare, partyId, partycount, threshold)
				if err != nil {
					return err
				}
				fmt.Printf("%s\n", local)
			}
			return nil
		},
	}
}
