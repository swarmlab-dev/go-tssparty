package tssparty

import (
	"encoding/json"

	"github.com/bnb-chain/tss-lib/v2/eddsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/tss"
)

func NewEddsaKeygenTssParty(localID string, n int, t int) KeygenTssParty {
	return &EddsaKeygenTssPartyState{
		tssPartyState: NewTssPartyState(NewPartyID(localID, nil), n, t),
	}
}

func NewEddsaKeygenTssPartyWithKey(localID string, localKeyBase16 string, n int, t int) (KeygenTssParty, error) {
	partyId, err := NewPartyIDFromKey(localID, localKeyBase16, 16)
	if err != nil {
		return nil, err
	}
	return &EddsaKeygenTssPartyState{
		tssPartyState: NewTssPartyState(partyId, n, t),
	}, nil
}

func (party *EddsaKeygenTssPartyState) GetKeyShare() (string, error) {
	return party.stateFunc2(PEERS_KNOWN, TSS_DONE, func() (string, error) {
		// init keygen party
		outCh := make(chan tss.Message)
		endCh := make(chan *keygen.LocalPartySaveData)
		defer close(outCh)
		defer close(endCh)
		tssParams := party.GetParams(true)
		eddsaKeygenParty := keygen.NewLocalParty(tssParams, outCh, endCh)

		// start
		go party.ProcessOutgoingMessageToTransport(outCh)
		go party.ProcessIncomingMessageFromTransport(eddsaKeygenParty)
		errp := eddsaKeygenParty.Start()
		if errp != nil {
			return "", errp
		}

		// return generated key share
		ret := <-endCh
		jsonRet, err := json.Marshal(ret)
		if err != nil {
			return "", err
		}
		return string(jsonRet), nil
	})
}

func JsonToEddsaKey(jsonEddsaKey string) (*keygen.LocalPartySaveData, error) {
	var key keygen.LocalPartySaveData
	err := json.Unmarshal([]byte(jsonEddsaKey), &key)
	if err != nil {
		return nil, err
	}
	return &key, nil
}
