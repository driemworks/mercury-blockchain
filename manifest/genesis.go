package manifest

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
        "0x9F0d31dFE801cc74ED9e50F06aDC7B168FF2F35b": {
			"alias": "tony",
            "sent": [],
            "inbox": [],
			"balance": 100000000,
			"pending_balance": 100000000
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
