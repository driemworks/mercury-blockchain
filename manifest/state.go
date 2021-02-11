package manifest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"

	"github.com/ethereum/go-ethereum/common"
)

type CID string

type SentItem struct {
	To   common.Address `json:"to"`
	CID  CID            `json:"cid"`
	Hash Hash           `json:"hash"`
}

type InboxItem struct {
	From common.Address `json:"from"`
	CID  CID            `json:"cid"`
	Hash Hash           `json:"hash"`
}

type Manifest struct {
	Sent           []SentItem  `json:"sent"`
	Inbox          []InboxItem `json:"inbox"`
	Balance        float32     `json:"balance"`
	PendingBalance float32     `json:"pending_balance"`
}

type State struct {
	Manifest        map[common.Address]Manifest
	Account2Nonce   map[common.Address]uint
	txMempool       []Tx
	latestBlock     Block
	latestBlockHash Hash
	dbFile          *os.File
	hasGenesisBlock bool
}

/*
* Loads the current state by replaying all transactions in tx.db
* on top of the gensis state as defined in genesis.json
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
	manifest := make(map[common.Address]Manifest)
	for account, mailbox := range gen.Manifest {
		fmt.Printf("the account is %x\n", account)
		fmt.Printf("the account balance is %x\n", mailbox.Balance)
		manifest[account] = Manifest{mailbox.Sent, mailbox.Inbox, mailbox.Balance, mailbox.PendingBalance}
	}

	blockDbFile, err := os.OpenFile(getBlocksDbFilePath(datadir), os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(blockDbFile)
	account2Nonce := make(map[common.Address]uint)
	state := &State{manifest, account2Nonce, make([]Tx, 0), Block{}, Hash{}, blockDbFile, false}
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
		if err = applyBlock(blockFs.Value, state); err != nil {
			return nil, err
		}
		state.latestBlock = blockFs.Value
		state.latestBlockHash = blockFs.Key
		state.hasGenesisBlock = true
	}
	return state, nil
}

// func (s *State) AddBlocks(blocks []Block) error {
// 	for _, b := range blocks {
// 		_, err := s.AddBlock(b)
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }

func (s *State) AddBlock(b Block) (Hash, error) {
	pendingState := s.copy()

	err := applyBlock(b, &pendingState)
	if err != nil {
		return Hash{}, err
	}

	blockHash, err := b.Hash()
	if err != nil {
		return Hash{}, err
	}

	blockFs := BlockFS{blockHash, b}

	blockFsJson, err := json.Marshal(blockFs)
	if err != nil {
		return Hash{}, err
	}

	fmt.Printf("Persisting new Block to disk:\n")
	fmt.Printf("\t%s\n", blockFsJson)

	_, err = s.dbFile.Write(append(blockFsJson, '\n'))
	if err != nil {
		return Hash{}, err
	}

	s.Account2Nonce = pendingState.Account2Nonce
	s.Manifest = pendingState.Manifest
	s.latestBlockHash = blockHash
	s.latestBlock = b
	s.hasGenesisBlock = true

	return blockHash, nil
}

func (s *State) NextBlockNumber() uint64 {
	if !s.hasGenesisBlock {
		return uint64(0)
	}
	return s.LatestBlock().Header.Number + 1
}

/**
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
* 1) appends a sent item to the tx's from account
* 2) appends an inbox item to t he tx's to account
 */
func applyTx(tx SignedTx, s *State) error {
	ok, err := tx.IsAuthentic()
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("bad Tx. Sender '%s' is forged", tx.From.String())
	}

	expectedNonce := s.Account2Nonce[tx.From] + 1
	if tx.Nonce != expectedNonce {
		return fmt.Errorf("bad Tx. next nonce must be '%d', not '%d'", expectedNonce, tx.Nonce)
	}

	// TODO - what's happening here?
	fmt.Println(s.Manifest[tx.To].PendingBalance)
	fmt.Println(s.Manifest[tx.From])
	if s.Manifest[tx.From].PendingBalance < 1 {
		return fmt.Errorf("bad Tx. You have no remaining balance")
	}

	txHash, err := tx.Hash()
	if err != nil {
		return fmt.Errorf("bad Tx. Can't calculate tx hash")
	}

	// TODO update balances!!!!!
	// update sender balance and sent items
	var senderMailbox = s.Manifest[tx.From]
	senderMailbox.Sent = append(senderMailbox.Sent, SentItem{tx.To, tx.CID, txHash})
	senderMailbox.Balance = senderMailbox.PendingBalance
	s.Manifest[tx.From] = senderMailbox
	// update recipient inbox items
	var receipientMailbox = s.Manifest[tx.To]
	receipientMailbox.Inbox = append(receipientMailbox.Inbox, InboxItem{tx.From, tx.CID, txHash})
	s.Manifest[tx.To] = receipientMailbox
	tmp := s.Account2Nonce
	tmp[tx.From] = tx.Nonce
	s.Account2Nonce = tmp
	return nil
}

func NewCID(cid string) CID {
	return CID(cid)
}

// applyBlock verifies if block can be added to the blockchain.
//
// Block metadata are verified as well as transactions within (sufficient balances, etc).
func applyBlock(b Block, s *State) error {
	nextExpectedBlockNumber := s.latestBlock.Header.Number + 1

	if s.hasGenesisBlock && b.Header.Number != nextExpectedBlockNumber {
		return fmt.Errorf("next expected block must be '%d' not '%d'", nextExpectedBlockNumber, b.Header.Number)
	}

	if s.hasGenesisBlock && s.latestBlock.Header.Number > 0 && !reflect.DeepEqual(b.Header.Parent, s.latestBlockHash) {
		return fmt.Errorf("next block parent hash must be '%x' not '%x'", s.latestBlockHash, b.Header.Parent)
	}

	hash, err := b.Hash()
	if err != nil {
		return err
	}
	if !IsBlockHashValid(hash) {
		return fmt.Errorf("Invalid block hash")
	}
	err = applyTXs(b.TXs, s)
	if err != nil {
		return err
	}

	// reward 1000 each time a tx is mined... does this seem like an excessive amount?
	tmp := s.Manifest[b.Header.Miner]
	tmp.Balance += 100
	tmp.PendingBalance += 100
	s.Manifest[b.Header.Miner] = tmp

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

func (s *State) LatestBlock() Block {
	return s.latestBlock
}

func (s *State) copy() State {
	copy := State{}
	copy.hasGenesisBlock = s.hasGenesisBlock
	copy.dbFile = s.dbFile
	copy.latestBlock = s.latestBlock
	copy.latestBlockHash = s.latestBlockHash
	copy.txMempool = make([]Tx, len(s.txMempool))
	copy.Manifest = make(map[common.Address]Manifest)
	copy.Account2Nonce = make(map[common.Address]uint)
	for account, manifest := range s.Manifest {
		copy.Manifest[account] = manifest
	}
	for account, nonce := range s.Account2Nonce {
		copy.Account2Nonce[account] = nonce
	}
	return copy
}

func GetBlocksAfter(blockHash Hash, dataDir string) ([]Block, error) {
	f, err := os.OpenFile(getBlocksDbFilePath(dataDir), os.O_RDONLY, 0600)
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
