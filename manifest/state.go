package manifest

import (
	"bufio"
	"encoding/json"
	"os"
	"time"
)

type CID string

type SentItem struct {
	To  Account `json:"to"`
	CID CID     `json:"cid"`
}

type InboxItem struct {
	From Account `json:"from"`
	CID  CID     `json:"cid"`
}

type Manifest struct {
	Sent  []SentItem
	Inbox []InboxItem
}

type State struct {
	Manifest        map[Account]Manifest
	txMempool       []Tx
	latestBlockHash Hash
	dbFile          *os.File
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
	// load the manifest -> consider refactoring..
	// using manifest as var and Manifest as type, but they are not the same thing
	manifest := make(map[Account]Manifest)
	for account, mailbox := range gen.Manifest {
		manifest[account] = Manifest{mailbox.Sent, mailbox.Inbox}
	}
	txFile, err := os.OpenFile(getBlocksDbFilePath(datadir), os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	// read lines in tx.db
	scanner := bufio.NewScanner(txFile)
	// load initial state
	state := &State{manifest, make([]Tx, 0), Hash{}, txFile}
	for scanner.Scan() {
		// handle scanner error
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		var blockFs BlockFS
		if err := json.Unmarshal(scanner.Bytes(), &blockFs); err != nil {
			return nil, err
		}
		if err = state.applyBlock(blockFs.Value); err != nil {
			return nil, err
		}
		state.latestBlockHash = blockFs.Key
	}
	return state, nil
}

func (s *State) applyBlock(b Block) error {
	for _, tx := range b.TXs {
		if err := s.apply(tx); err != nil {
			return err
		}
	}
	return nil
}

func (s *State) AddBlock(b Block) error {
	for _, tx := range b.TXs {
		if err := s.AddTx(tx); err != nil {
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
func (s *State) apply(tx Tx) error {
	// if it is a reward, just.. do nothing for now..
	if tx.IsReward() {
		return nil
	}
	// since there is no concept of a reward, coin, etc yet,
	// just add a fake CID for now
	// does not account for anything that could be happening as to the
	// sender receiving cid's
	var senderDirectory = s.Manifest[tx.From]
	senderDirectory.Sent = append(senderDirectory.Sent, SentItem{tx.To, tx.CID})
	s.Manifest[tx.From] = senderDirectory

	var receiverDirectory = s.Manifest[tx.To]
	receiverDirectory.Inbox = append(receiverDirectory.Inbox, InboxItem{tx.From, tx.CID})
	s.Manifest[tx.To] = receiverDirectory

	return nil
}

/*
* Add the transaction to the state
* 1) update tx sender/receiver state by calling apply
* 2) append tx to txMempool (to be mined later)
 */
func (s *State) AddTx(tx Tx) error {
	// try to apply the tx to the state
	if err := s.apply(tx); err != nil {
		return err
	}
	// append to the tx mempool (to be mined later)
	s.txMempool = append(s.txMempool, tx)
	return nil
}

func NewCID(cid string) CID {
	return CID(cid)
}

/*
* Persist the state's tx mempool to tx.db
 */
func (s *State) Persist() (Hash, error) {
	block := NewBlock(s.latestBlockHash, uint64(time.Now().Unix()), s.txMempool)
	blockHash, err := block.Hash()
	if err != nil {
		return Hash{}, err
	}

	blockFs := BlockFS{blockHash, block}
	blockFsJson, err := json.Marshal(blockFs)
	if err != nil {
		return Hash{}, err
	}
	if _, err = s.dbFile.Write(append(blockFsJson, '\n')); err != nil {
		return Hash{}, err
	}
	s.latestBlockHash = blockHash
	s.txMempool = []Tx{}
	return s.latestBlockHash, nil
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
