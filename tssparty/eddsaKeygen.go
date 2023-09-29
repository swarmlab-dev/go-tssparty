package tssparty

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/eddsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/swarmlab-dev/go-partybus/partybus"
)

func JoinEddsaKeygenParty(partyBusUrl string, sessionId string, partyId string, partycount int, threshold int) error {
	fmt.Printf("ecdsa keygen session %s: partyCount=%v threshold=%v\n", sessionId, partycount, threshold)
	key := common.MustGetRandomInt(256)
	thisParty := tss.NewPartyID(partyId, "", key)
	fmt.Printf("local party is %s (%s)", thisParty.Id, hex.EncodeToString(thisParty.Key))

	out := make(chan partybus.PeerMessage)
	in := make(chan partybus.PeerMessage)
	sig := make(chan partybus.StatusMessage)

	defer close(out)
	defer close(in)
	defer close(sig)
	err := partybus.ConnectToPartyBus(partyBusUrl, sessionId, thisParty.Id, out, in, sig)
	if err != nil {
		return err
	}

	// wait until all guests arrived at the party
	fmt.Printf("wait all guest before starting the party...\n")
	var guests []string
	for status := range sig {
		guests = status.Peers
		if len(guests) == partycount {
			break
		}
	}
	fmt.Printf("alright! %s got %v guests: [ %s ]\n", sessionId, partycount, strings.Join(guests, ", "))

	// share partyId and wait until all partyIds are received
	fmt.Printf("sharing keys...\n")
	parties := make([]*tss.PartyID, partycount)
	parties[0] = thisParty

	thisPartyJson, err := json.Marshal(thisParty)
	if err != nil {
		return err
	}
	out <- partybus.NewBroadcastMessage(thisParty.Id, thisPartyJson)

	i := 1
	for msg := range in {
		var peerPartyId tss.PartyID
		err := json.Unmarshal(msg.Msg, &peerPartyId)
		if err != nil {
			return err
		}

		if msg.From != peerPartyId.Id {
			return fmt.Errorf("partyId should be the same as message origin")
		}

		parties[i] = &peerPartyId
		i++

		if i == partycount {
			break
		}
	}

	// build threshold context
	sortedParties := tss.SortPartyIDs(parties)
	ctx := tss.NewPeerContext(sortedParties)
	ec := tss.S256()
	params := tss.NewParameters(ec, ctx, thisParty, len(sortedParties), threshold)
	partyIDMap := make(map[string]*tss.PartyID)
	for _, id := range sortedParties {
		partyIDMap[id.Id] = id
	}

	fmt.Printf("sorted parties: [ %s ]\n", strings.Join(MapArrayOfPartyID(sortedParties, func(p *tss.PartyID) string { return fmt.Sprintf("%s (%s)", p.Id, hex.EncodeToString(p.Key)) }), ", "))

	// start the party
	outCh := make(chan tss.Message)
	endCh := make(chan *keygen.LocalPartySaveData)
	party := keygen.NewLocalParty(params, outCh, endCh)

	// process outgoing messages to be send to peers
	go func(ch <-chan tss.Message) {
		for msg := range ch {
			bytes, _, err := msg.WireBytes()
			if err != nil {
				fmt.Printf("error while wiring message to peers: %s\n", err.Error())
				return
			}
			to := MapArrayOfPartyID(msg.GetTo(), func(p *tss.PartyID) string { return p.Id })
			fmt.Printf("%s >>> [ %s ] (%v)\n", thisParty.Id, strings.Join(to, ", "), len(bytes))
			out <- partybus.NewMulticastMessage(thisParty.Id, to, bytes)
		}
	}(outCh)

	// process incoming messages to be received from peers
	go func(ch <-chan partybus.PeerMessage) {
		for msg := range ch {
			fmt.Printf("[ %s ] <<< %s (%v)\n", strings.Join(msg.To, ", "), msg.From, len(msg.Msg))
			_, err := party.UpdateFromBytes(msg.Msg, partyIDMap[msg.From], true)
			if err != nil {
				fmt.Printf("error while receiving message from peer %s: %s\n", msg.From, err.Error())
				return
			}
		}
	}(in)

	go func() {
		err := party.Start()
		if err != nil {
			fmt.Printf("error while partying: %s\n", err.Error())
		}
	}()

	local := <-endCh
	jsonLocal, _ := json.Marshal(local)
	fmt.Printf("local share: %s", jsonLocal)

	return nil

}
