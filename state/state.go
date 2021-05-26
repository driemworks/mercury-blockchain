package state

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"

	"github.com/driemworks/mercury-blockchain/core"

	"github.com/ethereum/go-ethereum/common"
	"github.com/raphamorim/go-rainbow"
)

const BlockReward = float32(10)

type CurrentNodeState struct {
	Subscriptions  [][]byte `json:"subscriptions"`
	Channels       [][]byte `json:"channels"`
	Balance        float32  `json:"balance"`
	PendingBalance float32  `json:"pending_balance"`
}

type State struct {
	Catalog              map[common.Address]CurrentNodeState
	Account2Nonce        map[common.Address]uint
	PendingAccount2Nonce map[common.Address]uint
	txMempool            []Tx
	latestBlock          Block
	latestBlockHash      Hash
	dbFile               *os.File
	datadir              string
	hasGenesisBlock      bool
}

/*
* Loads the current state by replaying all transactions in block.db
* on top of the genesis state as defined in genesis.json
 */
func NewStateFromDisk(datadir string) (*State, error) {
	err := initDataDirIfNotExists(datadir)
	if err != nil {
		return nil, err
	}
	gen, err := loadGenesis(getGenesisJsonFilePath(datadir))
	if err != nil {
		return nil, err
	}
	// load the manifest -> consider refactoring name..
	// using manifest as var and Manifest as type, but they are not the same thing
	manifest := make(map[common.Address]CurrentNodeState)
	for account, s := range gen.State {
		manifest[account] = CurrentNodeState{s.Subscriptions, s.Channels, s.Balance, s.PendingBalance}
	}

	blockDbFile, err := os.OpenFile(getBlocksDbFilePath(datadir, false), os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(blockDbFile)
	account2Nonce := make(map[common.Address]uint)
	pendingAccount2Nonce := make(map[common.Address]uint)
	state := &State{manifest, account2Nonce, pendingAccount2Nonce, make([]Tx, 0), Block{}, Hash{}, blockDbFile, datadir, true}
	for scanner.Scan() {
		// handle scanner error
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		blockFsJSON := scanner.Bytes()
		if len(blockFsJSON) == 0 {
			break
		}
		var blockFs BlockFS
		if err := json.Unmarshal(blockFsJSON, &blockFs); err != nil {
			return nil, err
		}
		if err = ApplyBlock(blockFs.Value, state); err != nil {
			return nil, err
		}
		state.latestBlock = blockFs.Value
		state.latestBlockHash = blockFs.Key
	}
	return state, nil
}

func (s *State) AddBlock(b Block) (*State, Hash, error) {
	pendingState := s.copy()
	latestBlock := s.latestBlock
	if s.hasGenesisBlock && b.Header.Number < latestBlock.Header.Number+1 {
		// if s.latestBlock.Header.Number == b.Header.Number {
		return nil, Hash{}, nil
		// }
	}
	err := ApplyBlock(b, &pendingState)
	if err != nil {
		return nil, Hash{}, err
	}

	blockHash, err := b.Hash()
	if err != nil {
		return nil, Hash{}, err
	}

	blockFs := BlockFS{blockHash, b}

	blockFsJSON, err := json.Marshal(blockFs)
	if err != nil {
		return nil, Hash{}, err
	}

	prettyJSON, err := core.PrettyPrintJSON(blockFsJSON)
	fmt.Printf("Persisting new Block to disk:\n")
	fmt.Printf("\t%s\n", &prettyJSON)

	_, err = s.dbFile.Write(append(blockFsJSON, '\n'))
	if err != nil {
		return nil, Hash{}, err
	}

	s.Account2Nonce = pendingState.Account2Nonce
	s.Catalog = pendingState.Catalog
	s.latestBlockHash = blockHash
	s.latestBlock = b
	s.hasGenesisBlock = true

	return nil, blockHash, nil
}

func ApplyBlock(b Block, s *State) error {
	nextExpectedBlockNumber := s.latestBlock.Header.Number + 1
	hash, err := b.Hash()
	if err != nil {
		return err
	}

	if s.hasGenesisBlock && b.Header.Number != nextExpectedBlockNumber {
		return fmt.Errorf("next expected block number must be '%d' not '%d'", nextExpectedBlockNumber, b.Header.Number)
	} else if s.hasGenesisBlock && s.latestBlock.Header.Number > 0 && !reflect.DeepEqual(b.Header.Parent, s.latestBlockHash) {
		return fmt.Errorf("next block parent hash must be '%x' not '%x'", s.latestBlockHash, b.Header.Parent)
	}
	if !IsBlockHashValid(hash) {
		return fmt.Errorf(rainbow.Red("Invalid block hash %x"), hash)
	}
	err = applyTXs(b.TXs, s)
	if err != nil {
		return err
	}
	tmp := s.Catalog[b.Header.Miner]
	tmp.Balance += BlockReward
	tmp.PendingBalance += BlockReward
	s.Catalog[b.Header.Miner] = tmp

	return nil
}

func (s *State) NextBlockNumber() uint64 {
	if !s.hasGenesisBlock {
		return uint64(0)
	}
	return s.LatestBlock().Header.Number + 1
}

/*
*
 */
func applyTXs(txs []SignedTx, s *State) error {
	sort.Slice(txs, func(i, j int) bool {
		return txs[i].Time < txs[j].Time
	})

	for _, tx := range txs {
		err := applyTx(tx, s)
		if err != nil {
			return err
		}
	}

	return nil
}

/*
* apply the transaction to the current state
 */
func applyTx(tx SignedTx, s *State) error {
	ok, err := tx.IsAuthentic()
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("bad Tx. Sender '%s' is forged", tx.Author.String())
	}
	expectedNonce := s.Account2Nonce[tx.Author] + 1
	if tx.Nonce != expectedNonce {
		// this is a possible case of another miner mining the same block!
		return fmt.Errorf("bad Tx. next nonce must be '%d', not '%d'", expectedNonce, tx.Nonce)
	}

	// if s.Manifest[tx.From].PendingBalance < 1 {
	// 	return fmt.Errorf("bad Tx. You have no remaining balance")
	// }

	_, err = tx.Hash()
	if err != nil {
		return fmt.Errorf("bad Tx. Can't calculate tx hash")
	}
	var currentNodeState = s.Catalog[tx.Author]
	// for now, just assume topic creation only?
	currentNodeState.Channels = append(currentNodeState.Channels, []byte(tx.Topic))
	// TODO: assume for now that topic creation costs 1 coin
	currentNodeState.Balance = currentNodeState.Balance - 1
	s.Catalog[tx.Author] = currentNodeState
	s.Account2Nonce[tx.Author] = tx.Nonce
	return nil
}

/*
* Close the connection to the file
 */
func (s *State) Close() {
	s.dbFile.Close()
}

/*
* Get the latest block hash from the current state
 */
func (s *State) LatestBlockHash() Hash {
	return s.latestBlockHash
}

/*
* Get the latest block from the current state
 */
func (s *State) LatestBlock() Block {
	return s.latestBlock
}

/*
* Copy the state
 */
func (s *State) copy() State {
	copy := State{}
	copy.hasGenesisBlock = s.hasGenesisBlock
	copy.dbFile = s.dbFile
	copy.latestBlock = s.latestBlock
	copy.latestBlockHash = s.latestBlockHash
	copy.txMempool = make([]Tx, len(s.txMempool))
	copy.Catalog = make(map[common.Address]CurrentNodeState)
	copy.Account2Nonce = make(map[common.Address]uint)
	for account, manifest := range s.Catalog {
		copy.Catalog[account] = manifest
	}
	for account, nonce := range s.Account2Nonce {
		copy.Account2Nonce[account] = nonce
	}
	return copy
}

/*
 Get all blocks in 'datadir' whose parent is a child of the block with the given block hash
*/
func GetBlocksAfter(blockHash Hash, dataDir string) ([]Block, error) {
	// open block.db
	f, err := os.OpenFile(getBlocksDbFilePath(dataDir, false), os.O_RDONLY, 0600)
	if err != nil {
		return nil, err
	}
	blocks := make([]Block, 0)
	shouldStartCollecting := false
	if reflect.DeepEqual(blockHash, Hash{}) {
		shouldStartCollecting = true
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		var blockFs BlockFS
		err = json.Unmarshal(scanner.Bytes(), &blockFs)
		if err != nil {
			return nil, err
		}

		if shouldStartCollecting {
			blocks = append(blocks, blockFs.Value)
			continue
		}

		if blockHash == blockFs.Key {
			shouldStartCollecting = true
		}
	}

	return blocks, nil
}
