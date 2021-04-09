package core

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type PeerNode struct {
	Name        string         `json:"name"`
	IP          string         `json:"ip"`
	Port        uint64         `json:"port"`
	IsBootstrap bool           `json:"is_bootstrap"`
	Address     common.Address `json:"address"`
	Connected   bool
}

// func (pn *PeerNode) UnmarshalJSON(data []byte) error {
// 	var v PeerNode
// 	if err := json.Unmarshal(data, &v); err != nil {
// 		return err
// 	}
// 	pn.Address = v.Address
// 	// pn.Volume = v
// 	return nil
// }

func (p PeerNode) TcpAddress() string {
	return fmt.Sprintf("%s:%d", p.IP, p.Port)
}

func NewPeerNode(name string, ip string, port uint64, isBootstrap bool, address common.Address, connected bool) PeerNode {
	return PeerNode{name, ip, port, isBootstrap, address, connected}
}
