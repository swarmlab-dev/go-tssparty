package tssparty

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/swarmlab-dev/go-partybus/partybus"
)

func ConnectAndGetKeyShare(party KeygenTssParty, partyBusUrl string, sessionId string) (string, error) {
	defer party.Clean()

	err := party.Init()
	if err != nil {
		return "", err
	}

	_, err = party.PrepareTransport(partyBusUrl, sessionId, party.GetPartyCount())
	if err != nil {
		return "", err
	}

	ret, err := party.GetKeyShare()
	if err != nil {
		return "", err
	}
	return ret, nil
}

func ConnectAndSignMessage(party SigningTssParty, partyBusUrl string, sessionId string, msg string) (string, error) {
	defer party.Clean()

	err := party.Init()
	if err != nil {
		return "", err
	}

	_, err = party.PrepareTransport(partyBusUrl, sessionId, party.GetThreshold()+1)
	if err != nil {
		return "", err
	}

	ret, err := party.SignMessage(msg)
	if err != nil {
		return "", err
	}
	return ret, nil
}

func NewLocalParty(localID string, localKey *big.Int) *tss.PartyID {
	key := localKey
	if key == nil {
		key = common.MustGetRandomInt(256)
	}
	thisParty := tss.NewPartyID(localID, localID, key)
	logger.Infof("local party is %s (%s)", thisParty.Id, hex.EncodeToString(thisParty.Key))
	return thisParty
}

func NewTssPartyState(localParty *tss.PartyID, n int, t int) *tssPartyState {
	return &tssPartyState{
		step:      IDLE,
		thisParty: localParty,
		n:         n,
		t:         t,
	}
}

func (party *tssPartyState) GetPartyCount() int {
	return party.n
}

func (party *tssPartyState) GetThreshold() int {
	return party.t
}

func (party *tssPartyState) Init() error {
	return party.stateFunc(IDLE, INITIALIZED, func() error {
		return nil
	})
}

func (party *tssPartyState) PrepareTransport(partyBusUrl string, sessionId string, n int) (string, error) {
	err := party.ConnectToPartyBus(partyBusUrl, sessionId)
	if err != nil {
		party.step = ERROR
		return "", err
	}

	ret, err := party.WaitForGuestsAndExchangeIDs(n)
	if err != nil {
		return "", err
	}

	return ret, nil
}

func (party *tssPartyState) ConnectToPartyBus(partyBusUrl string, sessionId string) error {
	return party.stateFunc(INITIALIZED, CONNECTED_TO_BUS, func() error {
		party.outBus = make(chan partybus.PeerMessage)
		in, sig, err := partybus.ConnectToPartyBus(partyBusUrl, sessionId, party.thisParty.Id, party.outBus)
		if err != nil {
			return err
		}
		party.inBus = in
		party.sigBus = sig
		party.aboardBus = true
		return nil
	})
}

func (party *tssPartyState) DisconnectFromBus() error {
	party.aboardBus = false
	close(party.outBus)
	return nil
}

func (party *tssPartyState) WaitForGuestsAndExchangeIDs(n int) (string, error) {
	err := party.WaitForGuests(n)
	if err != nil {
		return "", err
	}

	ret, err := party.ExchangeIds(n)
	if err != nil {
		return "", err
	}

	return ret, nil
}

func (party *tssPartyState) WaitForGuests(n int) error {
	return party.stateFunc(CONNECTED_TO_BUS, PEERS_CONNECTED, func() error {
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
	})
}

func (party *tssPartyState) ExchangeIds(n int) (string, error) {
	return party.stateFunc2(PEERS_CONNECTED, PEERS_KNOWN, func() (string, error) {
		logger.Debug("exchanging party ids...")

		parties := make([]*tss.PartyID, n)
		parties[0] = party.thisParty

		thisPartyJson, err := json.Marshal(party.thisParty)
		if err != nil {
			return "", err
		}
		party.outBus <- partybus.NewBroadcastMessage(party.thisParty.Id, thisPartyJson)

		i := 1
		for msg := range party.inBus {
			var peerPartyId tss.PartyID
			err := json.Unmarshal(msg.Msg, &peerPartyId)
			if err != nil {
				return "", err
			}

			if msg.From != peerPartyId.Id {
				return "", fmt.Errorf("partyId should be the same as message origin")
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

		ret := strings.Join(MapArrayOfPartyID(party.sortedParties, func(p *tss.PartyID) string { return p.Id }), ",")
		logger.Debugf("sorted parties: [ %s ]", strings.Join(MapArrayOfPartyID(party.sortedParties, func(p *tss.PartyID) string { return fmt.Sprintf("%s (%s)", p.Id, hex.EncodeToString(p.Key)) }), ", "))
		return ret, nil
	})
}

func (party *tssPartyState) GetParams(useEdwardCurve bool) *tss.Parameters {
	ctx := tss.NewPeerContext(party.sortedParties)
	if useEdwardCurve {
		return tss.NewParameters(tss.Edwards(), ctx, party.thisParty, party.n, party.t)
	} else {
		return tss.NewParameters(tss.S256(), ctx, party.thisParty, party.n, party.t)
	}
}

func (party *tssPartyState) ProcessOutgoingMessageToTransport(outCh <-chan tss.Message) {
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

func (party *tssPartyState) ProcessIncomingMessageFromTransport(localParty tss.Party) {
	for msg := range party.inBus {
		_, err := localParty.UpdateFromBytes(msg.Msg, party.partyIDMap[msg.From], msg.IsBroadcast())
		if err != nil {
			logger.Errorf("error while receiving message from peer %s: %s", msg.From, err.Error())
			return
		}
	}
}

func (party *tssPartyState) Clean() error {
	if party.aboardBus {
		return party.DisconnectFromBus()
	}
	return nil
}

func MapArrayOfPartyID(vs []*tss.PartyID, f func(*tss.PartyID) string) []string {
	vsm := make([]string, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}
