package manifest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"ftp2p/main/logging"
	"os"
	"reflect"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/raphamorim/go-rainbow"
)

const BlockReward = float32(100)

type CID struct {
	CID         string `json:"cid"`
	IPFSGateway string `json:"ipfs_gateway"`
}

type SentItem struct {
	To     common.Address `json:"to"`
	CID    CID            `json:"cid"`
	Hash   Hash           `json:"hash"`
	Amount float32        `json:"amount"`
}

type InboxItem struct {
	From   common.Address `json:"from"`
	CID    CID            `json:"cid"`
	Hash   Hash           `json:"hash"`
	Amount float32        `json:"amount"`
}

type Manifest struct {
	Sent           []SentItem  `json:"sent"`
	Inbox          []InboxItem `json:"inbox"`
	Balance        float32     `json:"balance"`
	PendingBalance float32     `json:"pending_balance"`
}

type State struct {
	Manifest             map[common.Address]Manifest
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
		manifest[account] = Manifest{mailbox.Sent, mailbox.Inbox, mailbox.Balance, mailbox.PendingBalance}
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

func (s *State) AddBlock(b Block) (Hash, error) {
	pendingState := s.copy()
	// if it's the parent hash, do nothing
	if b.Header.Number == s.latestBlock.Header.Number-1 {
		return Hash{}, nil
	}

	err := ApplyBlock(b, &pendingState)
	if err != nil {
		return Hash{}, err
	}

	blockHash, err := b.Hash()
	if err != nil {
		return Hash{}, err
	}

	blockFs := BlockFS{blockHash, b}

	blockFsJSON, err := json.Marshal(blockFs)
	if err != nil {
		return Hash{}, err
	}

	prettyJSON, err := logging.PrettyPrintJSON(blockFsJSON)
	fmt.Printf("Persisting new Block to disk:\n")
	fmt.Printf("\t%s\n", &prettyJSON)

	_, err = s.dbFile.Write(append(blockFsJSON, '\n'))
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
		// this is a possible case of another miner mining the same block!
		return fmt.Errorf("bad Tx. next nonce must be '%d', not '%d'", expectedNonce, tx.Nonce)
	}

	if s.Manifest[tx.From].PendingBalance < 1 {
		return fmt.Errorf("bad Tx. You have no remaining balance")
	}

	txHash, err := tx.Hash()
	if err != nil {
		return fmt.Errorf("bad Tx. Can't calculate tx hash")
	}

	var senderMailbox = s.Manifest[tx.From]
	senderMailbox.Sent = append(senderMailbox.Sent, SentItem{tx.To, tx.CID, txHash, tx.Amount})
	senderMailbox.Balance = senderMailbox.PendingBalance
	s.Manifest[tx.From] = senderMailbox
	// update recipient inbox items
	var receipientMailbox = s.Manifest[tx.To]
	fmt.Printf("updating recipient's balance from %x to %x", receipientMailbox.Balance, receipientMailbox.Balance+tx.Amount)
	receipientMailbox.Balance += tx.Amount
	// add a new inbox item if there is a CID
	if !tx.CID.IsEmpty() {
		receipientMailbox.Inbox = append(receipientMailbox.Inbox, InboxItem{tx.From, tx.CID, txHash, tx.Amount})
	}
	s.Manifest[tx.To] = receipientMailbox
	s.Account2Nonce[tx.From] = tx.Nonce
	return nil
}

func (c *CID) IsEmpty() bool {
	return len(c.CID) == 0
}

func NewCID(cid string, gateway string) CID {
	return CID{cid, gateway}
}

func ApplyBlock(b Block, s *State) error {
	nextExpectedBlockNumber := s.latestBlock.Header.Number + 1
	if s.hasGenesisBlock && b.Header.Number != nextExpectedBlockNumber {
		// scenario: we mined the same block as the incoming block
		//			1) check if b and s.latestBlock are the same block (same block number -> should check tx too maybe?)
		if s.latestBlock.Header.Number == b.Header.Number {
			bHash, err := b.Hash()
			if err != nil {
				return err
			}
			fmt.Printf("the block's hash is %x", rainbow.Bold(rainbow.BgMagenta(fmt.Sprint(bHash))))
			// are they the same block? compare the Proof of Work  of both blocks
			// block with greatest pow is added to blocks other block is orphaned or ignored
			if s.latestBlock.Header.PoW < b.Header.PoW {
				s.orphanLatestBlock()
				s, err := NewStateFromDisk(s.datadir)
				if err != nil {
					return err
				}
				fmt.Printf("Reloaded state. Latest blockhash is: %x", rainbow.Yellow(fmt.Sprint(s.latestBlockHash)))
			} else {
				return fmt.Errorf("next expected block number must be '%d' not '%d'", nextExpectedBlockNumber, b.Header.Number)
			}

		} else {
			return fmt.Errorf("next expected block number must be '%d' not '%d'", nextExpectedBlockNumber, b.Header.Number)
		}
	} else if s.hasGenesisBlock && s.latestBlock.Header.Number > 0 && !reflect.DeepEqual(b.Header.Parent, s.latestBlockHash) {
		return fmt.Errorf("next block parent hash must be '%x' not '%x'", s.latestBlockHash, b.Header.Parent)
	}

	hash, err := b.Hash()
	if err != nil {
		return err
	}
	if !IsBlockHashValid(hash) {
		return fmt.Errorf(rainbow.Red("Invalid block hash %x"), hash)
	}
	err = applyTXs(b.TXs, s)
	if err != nil {
		return err
	}
	fmt.Printf("Giving block reward to %x", rainbow.Yellow(b.Header.Miner.Hex()))
	tmp := s.Manifest[b.Header.Miner]
	tmp.Balance += BlockReward
	tmp.PendingBalance += BlockReward
	s.Manifest[b.Header.Miner] = tmp

	return nil
}

func (s *State) orphanLatestBlock() error {
	writeEmptyBlocksDbToDisk(getBlocksDbFilePath(s.datadir, true))
	tempDbFile, err := os.OpenFile(getBlocksDbFilePath(s.datadir, true), os.O_APPEND|os.O_RDWR, 0600)
	// ioutil.WriteFile(getBlocksDbFilePath(s.datadir, true), []byte(""), os.ModePerm)
	blockDbFile, err := os.OpenFile(s.dbFile.Name(), os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(blockDbFile)
	for scanner.Scan() {
		// handle scanner error
		if err := scanner.Err(); err != nil {
			return err
		}
		blockFsJSON := scanner.Bytes()
		if len(blockFsJSON) == 0 {
			break
		}
		var blockFs BlockFS
		if err := json.Unmarshal(blockFsJSON, &blockFs); err != nil {
			return err
		}
		// if the block's number equals the input block's number, then do nothing
		if blockFs.Value.Header.Number < s.latestBlock.Header.Number {
			tempDbFile.Write(append(blockFsJSON, '\n'))
		}
	}
	err = Rename(tempDbFile.Name(), blockDbFile.Name())
	if err != nil {
		return err
	}
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
