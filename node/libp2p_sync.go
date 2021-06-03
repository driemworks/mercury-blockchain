package node

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/driemworks/mercury-blockchain/core"
	"github.com/driemworks/mercury-blockchain/state"
	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"
	noise "github.com/libp2p/go-libp2p-noise"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

const (
	DiscoveryServiceTag            = "mercury-service-tag"
	DiscoveryServiceTag_PendingTxs = "pending_txs"
	DiscoveryServiceTag_Blocks     = "blocks"
	DiscoveryServiceTag_Announce   = "announce"
)

/*
	Build a libp2p host
*/
func makeHost(ip string, port int, insecure bool) (host.Host, error) {
	r := rand.Reader
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, err
	}
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/%s/tcp/%d", ip, port)),
		libp2p.Identity(priv),
		libp2p.DisableRelay(),
		libp2p.Security(noise.ID, noise.New),
		libp2p.EnableNATService(),
	}
	if insecure {
		opts = append(opts, libp2p.NoSecurity)
	}
	host, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		return nil, err
	}
	hostAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ipfs/%s", host.ID().Pretty()))
	addr := host.Addrs()[0]
	fullAddr := addr.Encapsulate(hostAddr)
	logrus.Infof("I am: %s\n", fullAddr)
	return host, nil
}

/*
	Manually add a peer to the DHT
	If doRelay = true then open a connection with the peer
*/
func addPeers(ctx context.Context, n Node, peersArg string, doRelay bool) {
	if len(peersArg) == 0 {
		return
	}
	peerStrs := strings.Split(peersArg, ",")
	for i := 0; i < len(peerStrs); i++ {
		peerID, peerAddr := MakePeer(peerStrs[i])
		n.host.Peerstore().AddAddr(peerID, peerAddr, peerstore.PermanentAddrTTL)
		if doRelay {
			peerinfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
			if err != nil {
				log.Fatalln(err)
			}
			err = n.host.Connect(ctx, *peerinfo)
			if err != nil {
				log.Fatalln(err)
			}
			s, err := n.host.NewStream(ctx, peerID, DiscoveryServiceTag_Announce)
			if err != nil {
				log.Fatalln(err)
			}
			hashBytes := []byte(n.state.LatestBlockHash().Hex())
			hashBytes = append(hashBytes, '\n')
			s.Write(hashBytes)
		}
	}
}

func (n *Node) runLibp2pNode(ctx context.Context, ip string, port int, bootstrapPeer string, name string) error {
	host, err := makeHost(ip, port, false)
	n.host = host
	if err != nil {
		return err
	}
	// add bootstrap nodes if provided
	if bootstrapPeer == "" {
		// TODO leaving as localhost for now. Should this be configurable?
		bootstrapPeer = fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", ip, port, host.ID().Pretty())
	} else {
		addPeers(ctx, *n, bootstrapPeer, true)
	}

	logrus.Infoln("Listening on", host.Addrs())
	logrus.Infoln("Protocols:", strings.Join(host.Mux().Protocols(), ", "))
	host.SetStreamHandler(DiscoveryServiceTag_Announce, func(s network.Stream) {
		// fmt.Println(DiscoveryServiceTag_Announce)
		// read the peer's latest blockhash from the stream
		buf := bufio.NewReader(s)
		bytes, err := buf.ReadBytes('\n')
		// decode bytes
		var decoded [32]byte
		hex.Decode(bytes[:], decoded[:])
		blocks, err := state.GetBlocksAfter(decoded, n.datadir)
		if err != nil {
			log.Fatalln(err)
		}
		if blocks != nil {
			streamData(ctx, host, DiscoveryServiceTag_Blocks, s.Conn().RemotePeer(), blocks)
		}
		if n.pendingTXs != nil {
			streamData(ctx, host, DiscoveryServiceTag_PendingTxs, s.Conn().RemotePeer(), n.pendingTXs)
		}
		err = s.Close()
		if err != nil {
			log.Fatalln(DiscoveryServiceTag_Announce, err)
		}
	})
	// sync pending txs (from bootstrap node) on startup
	host.SetStreamHandler(DiscoveryServiceTag_PendingTxs, func(s network.Stream) {
		// fmt.Println(DiscoveryServiceTag_PendingTxs)
		buf := bufio.NewReader(s)
		bytes, err := buf.ReadBytes('\n')
		if err != nil {
			log.Fatalln(err)
		}
		var txs map[string]state.SignedTx
		err = json.Unmarshal(bytes, &txs)
		if err != nil {
			log.Fatalln(err)
		}
		n.pendingTXs = txs
		err = s.Close()
		if err != nil {
			log.Fatalln(err)
		}
	})
	// sync blocks (from bootstrap) on startup
	host.SetStreamHandler(DiscoveryServiceTag_Blocks, func(s network.Stream) {
		// fmt.Println(DiscoveryServiceTag_Blocks)
		buf := bufio.NewReader(s)
		bytes, err := buf.ReadBytes('\n')
		if err != nil {
			log.Fatalln(err)
		}
		var blocks []state.Block
		err = json.Unmarshal(bytes, &blocks)
		if err != nil {
			log.Fatalln(err)
		}
		for _, b := range blocks {
			_, _, err = n.state.AddBlock(b)
			if err != nil {
				log.Fatalln(err)
			}
		}
		err = s.Close()
		if err != nil {
			log.Fatalln(err)
		}
	})

	peerinfo, err := peer.AddrInfoFromP2pAddr(multiaddr.StringCast(bootstrapPeer))
	tracer, err := pubsub.NewRemoteTracer(ctx, host, *peerinfo)
	if err != nil {
		panic(err)
	}
	// create a pubsub service using the GossipSub router
	var ps *pubsub.PubSub
	ps, err = pubsub.NewGossipSub(ctx, host, pubsub.WithEventTracer(tracer))
	n.pubsub = ps
	if err != nil {
		log.Fatalln(err)
	}
	go n.Join(ctx, core.PENDING_TX_TOPIC, 128, func(data *pubsub.Message) {
		var tx state.SignedTx
		err := json.Unmarshal(data.Data, &tx)
		if err != nil {
			logrus.Errorln("failed to unmarshal json to SignedTx: ", err)
		}
		n.AddPendingTX(tx)
	}, n.newPendingTXs)
	// join the reserved block sync topic
	go n.Join(ctx, core.NEW_BLOCKS_TOPIC, 128, func(data *pubsub.Message) {
		var b state.Block
		err := json.Unmarshal(data.Data, &b)
		if err != nil {
			log.Fatalln("failed to unmarshal json to Block: ", err)
		}
		s, _, err := n.state.AddBlock(b)
		if err != nil {
			if s != nil {
				n.state = s
			}
			logrus.Errorln("failed to add block: ", err)
		}
	}, n.newMinedBlocks)
	select {}
}

func streamData(ctx context.Context, host host.Host, topic protocol.ID, peerId peer.ID, data interface{}) {
	s, err := host.NewStream(ctx, peerId, topic)
	if err != nil {
		log.Fatalln(err)
	}
	dataJson, err := json.Marshal(data)
	if err != nil {
		errBytes, _ := json.Marshal(err)
		s.Write(errBytes)
	} else {
		dataJson = append(dataJson, '\n')
		s.Write(dataJson)
	}
}
