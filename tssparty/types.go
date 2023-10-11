package tssparty

import (
	"fmt"

	ecdsaKeygen "github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	eddsaKeygen "github.com/bnb-chain/tss-lib/v2/eddsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/swarmlab-dev/go-partybus/partybus"
)

type TssParty interface {
	GetPartyCount() int
	GetThreshold() int

	Init() error                                                  // step 1
	ConnectToPartyBus(partyBusUrl string, sessionId string) error // step 2
	WaitForGuests(n int) error                                    // step 3
	ExchangeIds(n int) (string, error)                            // step 4

	PrepareTransport(partyBusUrl string, sessionId string, n int) (string, error) // step 2, 3, 4
	WaitForGuestsAndExchangeIDs(n int) (string, error)                            // step 3, 4
	DisconnectFromBus() error
	Clean() error
}

type KeygenTssParty interface {
	TssParty
	GetKeyShare() (string, error) // step 5
}

type SigningTssParty interface {
	TssParty
	SignMessage(msg string) (string, error) // step 5
}

type tssPartyStep int64

const (
	IDLE             tssPartyStep = 0
	INITIALIZED      tssPartyStep = 1
	CONNECTED_TO_BUS tssPartyStep = 2
	PEERS_CONNECTED  tssPartyStep = 3
	PEERS_KNOWN      tssPartyStep = 4
	TSS_DONE         tssPartyStep = 5
	ERROR            tssPartyStep = 10
)

type tssPartyState struct {
	step tssPartyStep

	// tss parameter
	thisParty *tss.PartyID
	n         int
	t         int

	// transport partybus channel
	aboardBus     bool
	outBus        chan partybus.PeerMessage
	inBus         chan partybus.PeerMessage
	sigBus        chan partybus.StatusMessage
	sortedParties []*tss.PartyID
	partyIDMap    map[string]*tss.PartyID
}

type EcdsaKeygenTssPartyState struct {
	*tssPartyState
	preParams *ecdsaKeygen.LocalPreParams
}

type EddsaKeygenTssPartyState struct {
	*tssPartyState
}

type EcdsaSigningTssPartyState struct {
	*tssPartyState
	keyShare *ecdsaKeygen.LocalPartySaveData
}

type EddsaSigningTssPartyState struct {
	*tssPartyState
	keyShare *eddsaKeygen.LocalPartySaveData
}

// helper functions

func (party *tssPartyState) stateFunc(from tssPartyStep, to tssPartyStep, fun func() error) error {
	if err := party.checkState(from); err != nil {
		return err
	}

	if err := fun(); err != nil {
		return err
	}

	party.setState(to)
	return nil
}

func (party *tssPartyState) stateFunc2(from tssPartyStep, to tssPartyStep, fun func() (string, error)) (string, error) {
	if err := party.checkState(from); err != nil {
		return "", err
	}

	str, err := fun()
	if err != nil {
		return "", err
	}

	party.setState(to)
	return str, nil
}

func (party *tssPartyState) checkState(expected tssPartyStep) error {
	if party.step != expected {
		return fmt.Errorf("expected to be step %v but currently at step %v", expected, party.step)
	}
	return nil
}

func (party *tssPartyState) setState(step tssPartyStep) {
	switch step {
	case INITIALIZED:
		logger.Info("tssParty Initialized")
	case CONNECTED_TO_BUS:
		logger.Info("successfully connected to party bus")
	case PEERS_CONNECTED:
		logger.Info("all peers connected")
	case PEERS_KNOWN:
		logger.Info("all peer's ids are exchanged")
	case TSS_DONE:
		logger.Info("tss ceremony ended")
	case ERROR:
		logger.Info("party has errored")
	}
	party.step = step
}
