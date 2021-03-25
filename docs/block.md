# NOTE: THIS IS ALL BRAINSTORMING... not all accurate or coherent
# Block

## Overview
Each block consists of a header and a collection of signed transactions.

```
type Block struct {
	Header BlockHeader
	TXs    []SignedTx
}
```

where the header is:

```
type BlockHeader struct {
	Parent Hash           `json:"parent"`
	Time   uint64         `json:"time"`
	Number uint64         `json:"number"`
	Nonce  uint32         `json:"nonce"`
	Miner  common.Address `json:"miner"`
	PoW    int            `json:"proof_of_work"`
}
```

## Transactions
A transaction is an append only operation on the blockchain initiated by a single user.

Users can create pending transactions by calling the `/publish` and `/add-peer/` endpoints.

**DANGER**: What if I want to add more functionality in the future? new endpoints, new data poinst as part of the state, etc. Do I need to fork the blockchain? Is there a way to do it without having to do a fork? Figure it out later...

A transaction represents an action taken by a user that mutates the state of the system.
E.g. A user wants to upload an mp4 to the network. In that case, a transaction would look like:

```
{
    "time": "xyz...", // epoch time in millis
    "nonce": "", // [32]byte
    "publisher": "0x...", // address
    "action": "publish" or "add_peer",
    "payload": action_type_payload_instance
}
```
The `payload` field depends on which action type is provided.
## Action type payloads

### PUBLISH
[insert link to api doc request body]
```
{
    "type": "p2p" or "p2e",
    "cid": {
        "cid": "Qm...",
        "ipfs_gateway": "ipfs.io"
    },
    "to": "0x..."
}
```

### ADD_PEER
[insert link to api doc request body]
```
{
    "peer_address": "0x..."
}
```
When a node adds a pending transaction, it is enriched with the publishing node's address as well


I need to determine what transaction types should be...
in order to do so, let's look at each node's state:
```
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
```
The manifest is a map between addresses and a Manifest struct:
```
type Manifest struct {
	Sent           []SentItem  `json:"sent"`
	Inbox          []InboxItem `json:"inbox"`
	Balance        float32     `json:"balance"`
	PendingBalance float32     `json:"pending_balance"`
}
```
originally the idea was for this manifest to act as a mailbox, but I'm not doing that anymore...
instead I need to represent everything that a user would ever need to see...
NO! Here's a better idea:
    Create a "ftp2p query language".
    I.e. I want to be able to query my blockchain to retrieve data such as:
    "give me all events (transactions) between 01/01/2022 and 01/01/2023 published by address 0x..."
    https://thedevsaddam.medium.com/query-json-data-using-golang-76b6ab974dd6
    BOOM
 - this means we won't need to 'Manifest' object... but how should the state be constructed then?
 - i.e. how should the application work...

Example actions ("write operations"):
- adding a trusted peer (p2p)
- remove a trusted peer (p2p)
- publishing a public file (p2e)
- publishing a direct file (p2p)
- publishing a private file (p2Nil) -> really using ftp2p as a pinning service

1. assume each 'tx' is an append only operation -> i.e. insert only
2. pin a cid -> can have a recipient or not. if it does not, assume it is publicly accessible, validate public gateway?
3. things sent 'to you' will have you as the recipient... these are just cids, nothings too crazy... but what about trusting a peer? ('friend request')
4. want there to be a very 'live' aspect to this.. real time everything