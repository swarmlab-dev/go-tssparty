package tssparty

import (
	"encoding/json"

	eddsa "github.com/bnb-chain/tss-lib/v2/eddsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/tss"
)

func JoinEddsaKeygenParty(
	partyBusUrl string,
	sessionId string,
	partyId string,
	partycount int,
	threshold int) (string, error) {
	logger.Infof("eddsa keygen session %s: partyCount=%v threshold=%v", sessionId, partycount, threshold)

	// connect to bus and get all peer's ID
	tssParty := CreateNewTssParty(partycount, threshold, partyId, nil)
	tssParty.ConnectToBus(partyBusUrl, sessionId)
	defer tssParty.DisconnectFromBus()

	err := tssParty.WaitForGuests(partycount)
	if err != nil {
		return "", err
	}

	// init keygen party
	outCh := make(chan tss.Message)
	endCh := make(chan *eddsa.LocalPartySaveData)
	defer close(outCh)
	defer close(endCh)
	tssParams := tssParty.GetParams(true)
	eddsaKeygenParty := eddsa.NewLocalParty(tssParams, outCh, endCh)

	// start
	go tssParty.ProcessOutgoingMessageToTransport(outCh)
	go tssParty.ProcessIncomingMessageFromTransport(eddsaKeygenParty)
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
}
