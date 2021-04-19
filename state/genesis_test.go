package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_loadGenesis_Success(t *testing.T) {
	filepath := "../test/.mercury/genesis.json"
	genesis, err := loadGenesis(filepath)
	assert.Nil(t, err)
	assert.NotNil(t, genesis)
}
