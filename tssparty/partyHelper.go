package tssparty

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/bnb-chain/tss-lib/v2/common"
	ecdsaKeygen "github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	eddsaKeygen "github.com/bnb-chain/tss-lib/v2/eddsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/swarmlab-dev/go-partybus/partybus"
)

type TssParty struct {
	thisParty     *tss.PartyID
	n             int
	t             int
	sortedParties []*tss.PartyID
	partyIDMap    map[string]*tss.PartyID

	// transport partybus
	outBus chan partybus.PeerMessage
	inBus  chan partybus.PeerMessage
	sigBus chan partybus.StatusMessage
}

func CreateNewTssParty(n int, t int, localID string, localKey *big.Int) *TssParty {
	key := localKey
	if key == nil {
		key = common.MustGetRandomInt(256)
	}
	thisParty := tss.NewPartyID(localID, localID, key)
	logger.Infof("local party is %s (%s)", thisParty.Id, hex.EncodeToString(thisParty.Key))

	return &TssParty{
		thisParty: thisParty,
		n:         n,
		t:         t,
	}

}

func (party *TssParty) ConnectToBus(partyBusUrl string, sessionId string) error {
	party.outBus = make(chan partybus.PeerMessage)
	in, sig, err := partybus.ConnectToPartyBus(partyBusUrl, sessionId, party.thisParty.Id, party.outBus)
	if err != nil {
		return err
	}
	party.inBus = in
	party.sigBus = sig
	logger.Debugf("successfully connected to party bus")
	return nil
}

func (party *TssParty) DisconnectFromBus() error {
	close(party.outBus)
	return nil
}

func (party *TssParty) WaitForGuests(n int) error {
	err := party.WaitForAllPeers(n)
	if err != nil {
		return err
	}

	err = party.SharePartyIds(n)
	if err != nil {
		return err
	}

	return nil
}

func (party *TssParty) WaitForAllPeers(n int) error {
	logger.Debugf("wait for %v guest before starting the party...", party.n)

	var guests []string
	for status := range party.sigBus {
		guests = status.Peers
		if len(guests) == n {
			break
		}
	}

	if len(guests) != n {
		return fmt.Errorf("channel closed before all guests arrived")
	}

	logger.Debugf("party got %v guests: [ %s ]", n, strings.Join(guests, ", "))
	return nil
}

func (party *TssParty) SharePartyIds(n int) error {
	logger.Debug("exchanging party ids...")

	parties := make([]*tss.PartyID, n)
	parties[0] = party.thisParty

	thisPartyJson, err := json.Marshal(party.thisParty)
	if err != nil {
		return err
	}
	party.outBus <- partybus.NewBroadcastMessage(party.thisParty.Id, thisPartyJson)

	i := 1
	for msg := range party.inBus {
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

		if i == n {
			break
		}
	}

	party.sortedParties = tss.SortPartyIDs(parties)
	party.partyIDMap = make(map[string]*tss.PartyID)
	for _, id := range party.sortedParties {
		party.partyIDMap[id.Id] = id
	}

	logger.Debugf("sorted parties: [ %s ]", strings.Join(MapArrayOfPartyID(party.sortedParties, func(p *tss.PartyID) string { return fmt.Sprintf("%s (%s)", p.Id, hex.EncodeToString(p.Key)) }), ", "))
	return nil
}

func (party *TssParty) GetParams(useEdwardCurve bool) *tss.Parameters {
	ctx := tss.NewPeerContext(party.sortedParties)
	if useEdwardCurve {
		return tss.NewParameters(tss.Edwards(), ctx, party.thisParty, party.n, party.t)
	} else {
		return tss.NewParameters(tss.S256(), ctx, party.thisParty, party.n, party.t)
	}
}

func (party *TssParty) ProcessOutgoingMessageToTransport(outCh <-chan tss.Message) {
	for msg := range outCh {
		bytes, _, err := msg.WireBytes()
		if err != nil {
			logger.Errorf("error while wiring message to peers: %s", err.Error())
			return
		}
		to := MapArrayOfPartyID(msg.GetTo(), func(p *tss.PartyID) string { return p.Id })
		party.outBus <- partybus.NewMulticastMessage(party.thisParty.Id, to, bytes)
	}
}

func (party *TssParty) ProcessIncomingMessageFromTransport(localParty tss.Party) {
	for msg := range party.inBus {
		_, err := localParty.UpdateFromBytes(msg.Msg, party.partyIDMap[msg.From], msg.IsBroadcast())
		if err != nil {
			logger.Errorf("error while receiving message from peer %s: %s", msg.From, err.Error())
			return
		}
	}
}

func MapArrayOfPartyID(vs []*tss.PartyID, f func(*tss.PartyID) string) []string {
	vsm := make([]string, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}

func JsonToEcdsaKey(jsonEcdsaKey string) (*ecdsaKeygen.LocalPartySaveData, error) {
	var key ecdsaKeygen.LocalPartySaveData
	err := json.Unmarshal([]byte(jsonEcdsaKey), &key)
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func JsonToEddsaKey(jsonEddsaKey string) (*eddsaKeygen.LocalPartySaveData, error) {
	var key eddsaKeygen.LocalPartySaveData
	err := json.Unmarshal([]byte(jsonEddsaKey), &key)
	if err != nil {
		return nil, err
	}
	return &key, nil
}
