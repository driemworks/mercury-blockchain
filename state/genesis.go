package state

import (
	"encoding/json"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/common"
)

var genesisJson = `
{
    "genesis_time": "2021-02-012T00:00:00.000000000Z",
    "chain_id": "driemworks-blockchain",
    "manifest": {
        "0x96131b31b9935f6388502b502cf544c1a8c65ad6": {
			"alias": "tony",
            "sent": [],
            "inbox": [],
			"balance": 100000000,
			"pending_balance": 100000000
        }
    }
}`

type Genesis struct {
	Manifest map[common.Address]CurrentNodeState `json: "manifest"`
}

func loadGenesis(filepath string) (Genesis, error) {
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		return Genesis{}, err
	}

	var loadedGenesis Genesis
	err = json.Unmarshal(content, &loadedGenesis)
	if err != nil {
		return Genesis{}, err
	}
	return loadedGenesis, nil
}
