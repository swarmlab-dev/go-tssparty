package tssparty

import (
	"encoding/json"
	"time"

	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/tss"
)

func JoinEcdsaKeygenParty(
	partyBusUrl string,
	sessionId string,
	partyId string,
	partycount int,
	threshold int) (string, error) {
	logger.Infof("ecdsa keygen session %s: partyCount=%v threshold=%v\n", sessionId, partycount, threshold)

	// compute preparams
	logger.Debug("computing preparams...")
	preParams, _ := keygen.GeneratePreParams(1 * time.Minute)

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
	endCh := make(chan *keygen.LocalPartySaveData)
	defer close(outCh)
	defer close(endCh)
	tssParams := tssParty.GetParams(true)
	ecdsaKeygenParty := keygen.NewLocalParty(tssParams, outCh, endCh, *preParams)

	// start
	go tssParty.ProcessOutgoingMessageToTransport(outCh)
	go tssParty.ProcessIncomingMessageFromTransport(ecdsaKeygenParty)
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

}
