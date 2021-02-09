package manifest

import (
	"encoding/json"
	"io/ioutil"
)

var genesisJson = `
{
    "genesis_time": "2021-02-07T00:00:00.000000000Z",
    "chain_id": "driemworks-blockchain",
    "manifest": {
        "tony": {
            "sent": [],
            "inbox": [],
			"remainingTx": 100000
        }
    }
}`

type Genesis struct {
	Manifest map[Account]Manifest `json: "manifest"`
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
