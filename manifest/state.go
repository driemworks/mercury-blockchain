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
	state := &State{manifest, account2Nonce, pendingAccount2Nonce, make([]Tx, 0), Block{}, Hash{}, blockDbFile, datadir, false}
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
		state.hasGenesisBlock = true
	}
	return state, nil
}

func (s *State) AddBlock(b Block) (Hash, error) {
	pendingState := s.copy()

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
	senderMailbox.Sent = append(senderMailbox.Sent, SentItem{tx.To, tx.CID, txHash})
	senderMailbox.Balance = senderMailbox.PendingBalance
	s.Manifest[tx.From] = senderMailbox
	// update recipient inbox items
	var receipientMailbox = s.Manifest[tx.To]
	receipientMailbox.Balance = receipientMailbox.PendingBalance
	// add a new inbox item if there is a CID
	if tx.CID != NewCID("") {
		receipientMailbox.Inbox = append(receipientMailbox.Inbox, InboxItem{tx.From, tx.CID, txHash})
	}
	s.Manifest[tx.To] = receipientMailbox
	s.Account2Nonce[tx.From] = tx.Nonce
	return nil
}

func NewCID(cid string) CID {
	return CID(cid)
}

func ApplyBlock(b Block, s *State) error {
	nextExpectedBlockNumber := s.latestBlock.Header.Number + 1

	// if the incoming block was synced from a node that mined the same block before syncing
	// then the incoming block could possibly contain the same transactions as the currently mined block
	// TODO: what if the orphan block contains valid tx only partially not in the other block?
	// get the node's latest block
	// if s.latestBlock.Header.Number == b.Header.Number {
	// 	fmt.Println(rainbow.Red("Encountered two blocks with the same block number"))
	// 	// compare the pow of each block
	// 	if s.latestBlock.Header.PoW < b.Header.PoW {
	// 		fmt.Println("Reposessing mining reward. %s", rainbow.Bold(rainbow.Cyan("Sorry!")))
	// 		// 1) copy all but last line of block.db to block.db.tmp
	// 		// 2) rename block.db.tmp to block.db
	// 		// 3) rebuild state
	// 		// 4) create the (empty) tmp file
	// 		s.orphanLatestBlock()
	// 		newState, err := NewStateFromDisk(s.datadir)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		// will this work? ... let's find out
	// 		s = newState
	// 		// now can we just restart the node?
	// 		fmt.Printf(rainbow.Magenta("Successfully orphaned latest block"))
	// 	} else {
	// 		return fmt.Errorf("encountered invalid block. Rejecting it from the blockchain")
	// 	}
	// } else
	if s.hasGenesisBlock && b.Header.Number != nextExpectedBlockNumber {
		// scenario: we mined the same block as the incoming block
		//			1) check if they're the same block
		latestBlockHash, err := s.latestBlock.Hash()
		if err != nil {
			return err
		}
		blockHash, err := b.Hash()
		if err != nil {
			return err
		}
		if latestBlockHash == blockHash {
			// they're the same block! compare the Proof of Work  of both blocks
			// block with greatest pow is added to blocks other block is orphaned or ignored
			if s.latestBlock.Header.PoW < b.Header.PoW {
				s.orphanLatestBlock()
				s, err = NewStateFromDisk(s.datadir)
				if err != nil {
					return err
				}
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

	// if blockhash is empty, start collecting (i.e. append to blocks)
	// won't this always be true...? I hope...
	if reflect.DeepEqual(blockHash, Hash{}) {
		fmt.Println("shouldStartCollecting = true")
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

	if reflect.DeepEqual(blockHash, Hash{}) {
		fmt.Println("Ummm wait... we never found my block... that's weird")
		fmt.Println("I wonder if there could be another block with the same block number")
	}

	return blocks, nil
}
