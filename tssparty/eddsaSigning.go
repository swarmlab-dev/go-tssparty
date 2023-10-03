package tssparty

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/bnb-chain/tss-lib/v2/common"
	signing "github.com/bnb-chain/tss-lib/v2/eddsa/signing"
	"github.com/bnb-chain/tss-lib/v2/tss"
)

func JoinEddsaSigningParty(
	partyBusUrl string,
	sessionId string,
	message string,
	jsonKeyData string,
	partyId string,
	partycount int,
	threshold int) (string, error) {
	logger.Infof("ecdsa keygen session %s: partyCount=%v threshold=%v\n", sessionId, partycount, threshold)

	// get local key share
	key, err := JsonToEddsaKey(jsonKeyData)
	if err != nil {
		return "", err
	}

	// turn msg into a bigint
	msg := new(big.Int)
	if ret := msg.SetBytes([]byte(message)); ret == nil {
		return "", fmt.Errorf("cannot convert msg into a big int")
	}

	// connect to bus and get all peer's ID
	tssParty := CreateNewTssParty(partycount, threshold, partyId, key.ShareID)
	tssParty.ConnectToBus(partyBusUrl, sessionId)
	defer tssParty.DisconnectFromBus()

	err = tssParty.WaitForGuests(threshold + 1)
	if err != nil {
		return "", err
	}

	// init signing party
	outCh := make(chan tss.Message)
	endCh := make(chan *common.SignatureData)
	defer close(outCh)
	defer close(endCh)
	tssParams := tssParty.GetParams(true)
	eddsaSigningParty := signing.NewLocalParty(msg, tssParams, *key, outCh, endCh)

	// start
	go tssParty.ProcessOutgoingMessageToTransport(outCh)
	go tssParty.ProcessIncomingMessageFromTransport(eddsaSigningParty)
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
}
