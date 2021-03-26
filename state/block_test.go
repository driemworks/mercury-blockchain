package state

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_MarshallText(t *testing.T) {
	hash := buildValidHash()
	marshalledText, _ := hash.MarshalText()
	assert.Equal(t, byte(48), marshalledText[0])
}

func Test_UnMarshallText_InValid(t *testing.T) {
	hash := buildValidHash()
	dest := make([]byte, 32)
	err := hash.UnmarshalText(dest)
	assert.NotNil(t, err)
}

func Test_Hash_Valid(t *testing.T) {
	block := NewBlock(
		buildValidHash(),
		uint64(time.Now().Nanosecond()),
		1,
		nil,
		1,
		NewAddress("0x851EAF29553B0CE3180AC08c221050E2295D14cD"),
		100000,
	)
	blockHash, err := block.Hash()
	assert.Nil(t, err)
	// a new sha-256 hash will be generated each time.. there's not much we can do but assert its existence
	assert.NotNil(t, blockHash)
}

func Test_IsBlockHashValid_EmptyHash(t *testing.T) {
	invalidHash := Hash{}
	result := IsBlockHashValid(invalidHash)
	assert.False(t, result)
}

func Test_IsBlockHashValid_ValidHash(t *testing.T) {
	validHash := buildValidHash()
	result := IsBlockHashValid(validHash)
	assert.True(t, result)
}

func buildValidHash() Hash {
	validBytes := make([]byte, 32)
	validBytes[0] = byte(0)
	validBytes[1] = byte(0)
	validBytes[2] = byte(0)
	validBytes[3] = byte(9)
	var valid32Bytes [32]byte
	copy(valid32Bytes[:], validBytes[:])
	return Hash(valid32Bytes)
}
