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

func JoinEddsaKeygenParty(partyBusUrl string, sessionId string, partyId string, partycount int, threshold int) (*keygen.LocalPartySaveData, error) {
	logger.Infof("eddsa keygen session %s: partyCount=%v threshold=%v\n", sessionId, partycount, threshold)
	key := common.MustGetRandomInt(256)
	thisParty := tss.NewPartyID(partyId, "", key)
	logger.Infof("local party is %s (%s)\n", thisParty.Id, hex.EncodeToString(thisParty.Key))

	out := make(chan partybus.PeerMessage)
	in := make(chan partybus.PeerMessage)
	sig := make(chan partybus.StatusMessage)

	defer close(out)
	defer close(in)
	defer close(sig)
	err := partybus.ConnectToPartyBus(partyBusUrl, sessionId, thisParty.Id, out, in, sig)
	if err != nil {
		return nil, err
	}

	// wait until all guests arrived at the party
	logger.Debug("wait all guest before starting the party...")
	var guests []string
	for status := range sig {
		guests = status.Peers
		if len(guests) == partycount {
			break
		}
	}
	logger.Debugf("%s got %v guests: [ %s ]", sessionId, partycount, strings.Join(guests, ", "))

	// share partyId and wait until all partyIds are received
	logger.Debug("sharing keys...")
	parties := make([]*tss.PartyID, partycount)
	parties[0] = thisParty

	thisPartyJson, err := json.Marshal(thisParty)
	if err != nil {
		return nil, err
	}
	out <- partybus.NewBroadcastMessage(thisParty.Id, thisPartyJson)

	i := 1
	for msg := range in {
		var peerPartyId tss.PartyID
		err := json.Unmarshal(msg.Msg, &peerPartyId)
		if err != nil {
			return nil, err
		}

		if msg.From != peerPartyId.Id {
			return nil, fmt.Errorf("partyId should be the same as message origin")
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
	ec := tss.Edwards()
	params := tss.NewParameters(ec, ctx, thisParty, len(sortedParties), threshold)
	partyIDMap := make(map[string]*tss.PartyID)
	for _, id := range sortedParties {
		partyIDMap[id.Id] = id
	}

	logger.Debugf("sorted parties: [ %s ]", strings.Join(MapArrayOfPartyID(sortedParties, func(p *tss.PartyID) string { return fmt.Sprintf("%s (%s)", p.Id, hex.EncodeToString(p.Key)) }), ", "))

	// start the party
	outCh := make(chan tss.Message)
	endCh := make(chan *keygen.LocalPartySaveData)
	party := keygen.NewLocalParty(params, outCh, endCh)

	// process outgoing messages to be send to peers
	go func(ch <-chan tss.Message) {
		for msg := range ch {
			bytes, _, err := msg.WireBytes()
			if err != nil {
				logger.Error("error while wiring message to peers: %s", err.Error())
				return
			}
			to := MapArrayOfPartyID(msg.GetTo(), func(p *tss.PartyID) string { return p.Id })
			out <- partybus.NewMulticastMessage(thisParty.Id, to, bytes)
		}
	}(outCh)

	// process incoming messages to be received from peers
	go func(ch <-chan partybus.PeerMessage) {
		for msg := range ch {
			//fmt.Printf("[ %s ] <<< %s (%v)\n", strings.Join(msg.To, ", "), msg.From, len(msg.Msg))
			_, err := party.UpdateFromBytes(msg.Msg, partyIDMap[msg.From], msg.IsBroadcast())
			if err != nil {
				logger.Error("error while receiving message from peer %s: %s", msg.From, err.Error())
				return
			}
		}
	}(in)

	errp := party.Start()
	if errp != nil {
		return nil, errp
	}

	local := <-endCh
	return local, nil
}
