package tssparty

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/swarmlab-dev/go-partybus/partybus"
)

func JoinEcdsaKeygenParty(partyBusUrl string, sessionId string, partyId string, partycount int, threshold int) (*keygen.LocalPartySaveData, error) {
	logger.Infof("ecdsa keygen session %s: partyCount=%v threshold=%v\n", sessionId, partycount, threshold)
	key := common.MustGetRandomInt(256)
	thisParty := tss.NewPartyID(partyId, "", key)
	logger.Infof("local party is %s (%s)\n", thisParty.Id, hex.EncodeToString(thisParty.Key))

	// compute preparams
	logger.Debug("computing preparams...\n")
	preParams, _ := keygen.GeneratePreParams(1 * time.Minute)

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
	logger.Debug("wait for all guest before starting the party...\n")
	var guests []string
	for status := range sig {
		guests = status.Peers
		if len(guests) == partycount {
			break
		}
	}
	logger.Debug("%s got %v guests: [ %s ]\n", sessionId, partycount, strings.Join(guests, ", "))

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
			return nil, fmt.Errorf("partyId (%s) should be the same as websocket message origin (%s)", peerPartyId.Id, msg.From)
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

	logger.Debug("sorted parties: [ %s ]\n", strings.Join(MapArrayOfPartyID(sortedParties, func(p *tss.PartyID) string { return fmt.Sprintf("%s (%s)", p.Id, hex.EncodeToString(p.Key)) }), ", "))

	// start the party
	outCh := make(chan tss.Message)
	endCh := make(chan *keygen.LocalPartySaveData)
	party := keygen.NewLocalParty(params, outCh, endCh, *preParams)

	// process outgoing messages to be send to peers
	go func(ch <-chan tss.Message) {
		for msg := range ch {
			bytes, _, err := msg.WireBytes()
			if err != nil {
				logger.Error("error while wiring message to peers: %s\n", err.Error())
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
				logger.Error("error while updating party with message from %s: %s\n", msg.From, err.Error())
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

func MapArrayOfPartyID(vs []*tss.PartyID, f func(*tss.PartyID) string) []string {
	vsm := make([]string, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}
