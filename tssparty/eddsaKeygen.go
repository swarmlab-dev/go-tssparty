package tssparty

import (
	"encoding/json"
	"math/big"

	"github.com/bnb-chain/tss-lib/v2/eddsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/tss"
)

func NewEddsaKeygenTssParty(localID string, localKey *big.Int, n int, t int) KeygenTssParty {
	return &EddsaKeygenTssPartyState{
		tssPartyState: NewTssPartyState(NewLocalParty(localID, localKey), n, t),
	}
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
