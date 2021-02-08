package manifest

type Account string

func NewAccount(value string) Account {
	return Account(value)
}

type Tx struct {
	From Account `json: "from"`
	To   Account `json: "to"`
	CID  CID     `json: "cid"`
	Data string  `json: "data"`
}

func NewTx(from Account, to Account, cid CID, data string) Tx {
	return Tx{from, to, cid, data}
}

func (t Tx) IsReward() bool {
	// what would be a meaningful reward in the context of file sharing?
	// I suppose... the "in-app" currency maybe?
	// maybe there could be different tiers? based on number of transactions/day
	return t.Data == "reward"
}
