package manifest

import (
	"encoding/json"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/common"
)

var genesisJson = `
{
    "genesis_time": "2021-02-07T00:00:00.000000000Z",
    "chain_id": "driemworks-blockchain",
    "manifest": {
        "0x11494645b11e185ade0906d3a5f6b37b72a6c8dc": {
			"alias": "tony",
            "sent": [],
            "inbox": [],
			"balance": 100000,
			"pending_balance": 100000
        }
    }
}`

type Genesis struct {
	Manifest map[common.Address]Manifest `json: "manifest"`
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
