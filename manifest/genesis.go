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
        "0x7f3A1bB3B4b39838e9C2c20C74F2Be4a8b73d696": {
			"alias": "tony",
            "sent": [],
            "inbox": [],
			"balance": 100000,
			"pendingBalance": 100000
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

func writeGenesisToDisk(path string) error {
	return ioutil.WriteFile(path, []byte(genesisJson), 0644)
}
