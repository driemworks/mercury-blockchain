# Super Simple Proof of Stake
Mercury uses a proof of stake algorithm as details in a paper that will soon be added to this repo...

At a high level, it can be explained as:

1) A node stakes some amount of coins and gossips this to the network
   1) staking will be available via some rpc endpoint, or maybe command line arg? not sure yet... command line arg means you need to know how much you want to stake on startup, but maybe you don't know, or don't have tokens... I'll do an RPC endpoint
2) When some condition is met? (what does this mean?), nodes will choose a block creator
3) whenever a node wins an election, it creates a block and receives a reward (ranked reward for all who received votes?) -> can this translate to some concrete min/max apr? i.e. algorand