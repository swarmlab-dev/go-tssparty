package tssparty

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/tss"
)

func NewPartyID(localID string, localKey *big.Int) *tss.PartyID {
	key := localKey
	if key == nil {
		key = common.MustGetRandomInt(256)
	}
	thisParty := tss.NewPartyID(localID, localID, key)
	logger.Infof("local party is %s (%s)", thisParty.Id, hex.EncodeToString(thisParty.Key))
	return thisParty
}

func NewPartyIDFromKey(localID string, localKey string, base int) (*tss.PartyID, error) {
	key := new(big.Int)
	key, ok := key.SetString(localKey, base)
	if !ok {
		return nil, fmt.Errorf("cannot parse key string")
	}
	return tss.NewPartyID(localID, localID, key), nil
}
