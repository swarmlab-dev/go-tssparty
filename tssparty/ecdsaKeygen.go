package tssparty

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/tss"
)

func NewEcdsaKeygenTssParty(localID string, localKey *big.Int, n int, t int) KeygenTssParty {
	return &EcdsaKeygenTssPartyState{
		tssPartyState: NewTssPartyState(NewLocalParty(localID, localKey), n, t),
	}
}

func (party *EcdsaKeygenTssPartyState) Init() error {
	return party.stateFunc(IDLE, INITIALIZED, func() error {
		logger.Debug("computing preparams...")
		party.preParams, _ = keygen.GeneratePreParams(1 * time.Minute)
		return nil
	})
}

func (party *EcdsaKeygenTssPartyState) GetKeyShare() (string, error) {
	return party.stateFunc2(PEERS_KNOWN, TSS_DONE, func() (string, error) {
		outCh := make(chan tss.Message)
		endCh := make(chan *keygen.LocalPartySaveData)
		defer close(outCh)
		defer close(endCh)
		tssParams := party.GetParams(true)
		ecdsaKeygenParty := keygen.NewLocalParty(tssParams, outCh, endCh, *party.preParams)

		// start
		go party.ProcessOutgoingMessageToTransport(outCh)
		go party.ProcessIncomingMessageFromTransport(ecdsaKeygenParty)
		errp := ecdsaKeygenParty.Start()
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

func JsonToEcdsaKey(jsonEcdsaKey string) (*keygen.LocalPartySaveData, error) {
	var key keygen.LocalPartySaveData
	err := json.Unmarshal([]byte(jsonEcdsaKey), &key)
	if err != nil {
		return nil, err
	}
	return &key, nil
}
