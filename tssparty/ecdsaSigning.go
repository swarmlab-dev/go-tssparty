package tssparty

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/v2/tss"
)

func NewEcdsaSigningTssParty(localID string, jsonKeyShare string, n int, t int) (SigningTssParty, error) {
	key, err := JsonToEcdsaKey(jsonKeyShare)
	if err != nil {
		return nil, err
	}

	return &EcdsaSigningTssPartyState{
		tssPartyState: NewTssPartyState(NewPartyID(localID, key.ShareID), n, t),
		keyShare:      key,
	}, nil
}

func (party *EcdsaSigningTssPartyState) SignMessage(msgToSign string) (string, error) {
	return party.stateFunc2(PEERS_KNOWN, TSS_DONE, func() (string, error) {
		// turn msg into a bigint
		msg := new(big.Int)
		if ret := msg.SetBytes([]byte(msgToSign)); ret == nil {
			return "", fmt.Errorf("cannot convert msg into a big int")
		}

		outCh := make(chan tss.Message)
		endCh := make(chan *common.SignatureData)
		defer close(outCh)
		defer close(endCh)
		tssParams := party.GetParams(true)
		eddsaSigningParty := signing.NewLocalParty(msg, tssParams, *party.keyShare, outCh, endCh)

		// start
		go party.ProcessOutgoingMessageToTransport(outCh)
		go party.ProcessIncomingMessageFromTransport(eddsaSigningParty)
		errp := eddsaSigningParty.Start()
		if errp != nil {
			return "", errp
		}

		// return signed message
		ret := <-endCh
		jsonRet, err := json.Marshal(ret)
		if err != nil {
			return "", err
		}
		return string(jsonRet), nil
	})
}
