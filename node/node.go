package node

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"sync"
	"time"

	"github.com/driemworks/mercury-blockchain/core"
	pb "github.com/driemworks/mercury-blockchain/proto"
	"github.com/driemworks/mercury-blockchain/state"
	"github.com/multiformats/go-multiaddr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	mrand "math/rand"

	"github.com/ethereum/go-ethereum/common"
	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/raphamorim/go-rainbow"
)

// var logger = log.Logger{Prefix: "rendezvous"}

func handleStream(stream network.Stream) {

}

func makeRoutedHost(listenPort int, randseed int64, bootstrapPeers []peer.AddrInfo, globalFlag string) (host.Host, error) {

	// If the seed is zero, use real cryptographic randomness. Otherwise, use a
	// deterministic randomness source to make generated keys stay the same
	// across multiple runs
	var r io.Reader
	if randseed == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(randseed))
	}

	// Generate a key pair for this host. We will use it at least
	// to obtain a valid host ID.
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, err
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)),
		libp2p.Identity(priv),
		libp2p.DefaultTransports,
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
		libp2p.NATPortMap(),
	}

	ctx := context.Background()

	basicHost, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	// Construct a datastore (needed by the DHT). This is just a simple, in-memory thread-safe datastore.
	dstore := dsync.MutexWrap(ds.NewMapDatastore())

	// Make the DHT
	dht := dht.NewDHT(ctx, basicHost, dstore)

	// Make the routed host
	routedHost := rhost.Wrap(basicHost, dht)

	// connect to the chosen ipfs nodes
	err = bootstrapConnect(ctx, routedHost, bootstrapPeers)
	if err != nil {
		return nil, err
	}

	// Bootstrap the host
	err = dht.Bootstrap(ctx)
	if err != nil {
		return nil, err
	}

	// Build host multiaddress
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", routedHost.ID().Pretty()))

	// Now we can build a full multiaddress to reach this host
	// by encapsulating both addresses:
	// addr := routedHost.Addrs()[0]
	addrs := routedHost.Addrs()
	log.Println("I can be reached at:")
	for _, addr := range addrs {
		log.Println(addr.Encapsulate(hostAddr))
	}

	log.Printf("Now run \"./routed-echo -l %d -d %s%s\" on a different terminal\n", listenPort+1, routedHost.ID().Pretty(), globalFlag)

	return routedHost, nil
}

func (n *Node) RunLibP2P() {
	ctx := context.Background()
	r := mrand.New(mrand.NewSource(int64(n.port)))
	// Creates a new RSA key pair for this host.
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}
	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", n.ip, n.port))
	host, err := libp2p.New(
		ctx,
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(prvKey),
	)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Host created: %s", rainbow.Green(host.ID().Pretty()))
	fmt.Println()
	fmt.Println(host.Addrs())

	host.SetStreamHandler("/mercury/1.0.0", handleStream)
	kademliaDHT, err := dht.New(ctx, host)
	if err != nil {
		panic(err)
	}
	// bootstrap the dht
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}
	// connect to bootstrap nodes
	var wg sync.WaitGroup
	for _, peerAddr := range dht.DefaultBootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := host.Connect(ctx, *peerinfo); err != nil {
				fmt.Errorf("failed to establish connection with bootstrap", err)
			} else {
				fmt.Printf("Connection established with bootstrap node: %x", *peerinfo)
			}
			fmt.Println()
		}()
	}
	wg.Wait()
	// make yourself discoverable
	routingDiscovery := discovery.NewRoutingDiscovery(kademliaDHT)
	discovery.Advertise(ctx, routingDiscovery, "mercury")
	fmt.Println("Searching for peers...")
	peerChan, err := routingDiscovery.FindPeers(ctx, "mercury")
	for peer := range peerChan {
		if peer.ID == host.ID() {
			continue
		}
		fmt.Println()
		fmt.Printf("Found peer: %x", peer)
		fmt.Println()
		stream, err := host.NewStream(ctx, peer.ID, "/mercury/1.0.0")

		if err != nil {
			fmt.Errorf("Connection failed:", err)
			continue
		}
		_, err = stream.Write(n.info.Address.Bytes())
		if err != nil {
			fmt.Errorf("Failed to write to the stream", err)
			continue
		}
		out, err := ioutil.ReadAll(stream)
		fmt.Println(out)
		// } else {
		// sync pending transactions (using the stream)
		// rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
		// whenever there's a new pending transaction, push it to the stream
		// go writeData(n, rw)
		// go readData(rw)
		// }
	}
	select {}
}

const miningIntervalSeconds = 45
const syncIntervalSeconds = 30

type Node struct {
	password        string
	datadir         string
	ip              string
	port            uint64
	state           *state.State
	info            core.PeerNode
	knownPeers      map[string]core.PeerNode
	trustedPeers    map[string]core.PeerNode
	pendingTXs      map[string]state.SignedTx
	archivedTXs     map[string]state.SignedTx
	newSyncedBlocks chan state.Block
	newPendingTXs   chan state.SignedTx
	isMining        bool
	name            string
	tls             bool
}

func NewNode(name string, datadir string, ip string, port uint64,
	address common.Address, bootstrap core.PeerNode, password string, tls bool) *Node {
	knownPeers := make(map[string]core.PeerNode)
	knownPeers[bootstrap.TcpAddress()] = bootstrap
	return &Node{
		name:            name,
		datadir:         datadir,
		ip:              ip,
		port:            port,
		knownPeers:      knownPeers,
		trustedPeers:    make(map[string]core.PeerNode),
		info:            core.NewPeerNode(name, ip, port, false, address, true),
		pendingTXs:      make(map[string]state.SignedTx),
		archivedTXs:     make(map[string]state.SignedTx),
		newSyncedBlocks: make(chan state.Block),
		newPendingTXs:   make(chan state.SignedTx, 10000),
		isMining:        false,
		password:        password, // TODO this is temporary
		tls:             tls,
	}
}

/**
Run an RPC server to serve the implementation of the NodeServer
*/
func (n *Node) RunRPCServer(tls bool, certFile string, keyFile string) error {
	// start the server
	var opts []grpc.ServerOption
	if tls {
		tlsCredentials, err := loadTLSCredentials_Server()
		if err != nil {
			return err
		}
		opts = []grpc.ServerOption{grpc.Creds(tlsCredentials)}
	}
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterPublicNodeServer(grpcServer, newNodeServer(n))
	reflection.Register(grpcServer) // must register reflection api in order to invoke rpc externally
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", n.ip, 9090))
	// TODO expose this ip:port when DNS service built out?
	if err != nil {
		fmt.Printf("Could not listen on %s:%d", n.ip, 9090)
	}
	fmt.Println(fmt.Sprintf("Listening on: %s:%d", n.ip, 9090))
	err = grpcServer.Serve(lis)
	if err != nil {
		return err
	}
	return nil
}

func loadTLSCredentials_Server() (credentials.TransportCredentials, error) {
	serverCert, err := tls.LoadX509KeyPair("resources/cert/server-cert.pem", "resources/cert/server-key.pem")
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.NoClientCert,
	}
	return credentials.NewTLS(config), nil
}

func loadTLSCredentials_Client() (credentials.TransportCredentials, error) {
	pemServerCA, err := ioutil.ReadFile("resources/cert/ca-cert.pem")
	if err != nil {
		return nil, err
	}
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("failed to add server CA's certificate")
	}
	config := &tls.Config{
		RootCAs: certPool,
	}
	return credentials.NewTLS(config), nil
}

func RunRPCClient(ctx context.Context, tcp string, tls bool, caFile string, serverHostOverride string) (pb.PublicNodeClient, error) {
	var opts []grpc.DialOption
	if tls {
		tlsCreds, err := loadTLSCredentials_Client()
		if err != nil {
			log.Fatalf("cannot load TLS credentials: %s", err)
		}
		// NOTE: can add interceptors to add auth headers!
		opts = append(opts, grpc.WithTransportCredentials(tlsCreds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	opts = append(opts, grpc.WithBlock())
	conn, err := grpc.Dial(tcp, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
		return nil, err
	}
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()
	return pb.NewPublicNodeClient(conn), nil
}

func (n *Node) Run_RPC(ctx context.Context) error {
	// load the state
	state, err := state.NewStateFromDisk(n.datadir)
	if err != nil {
		return err
	}
	defer state.Close()
	n.state = state
	go n.RunLibP2P()
	// go n.sync(ctx)
	go n.mine(ctx)
	n.RunRPCServer(n.tls, "", "")
	return nil
}

func (n *Node) mine(ctx context.Context) error {
	var miningCtx context.Context
	var stopCurrentMining context.CancelFunc

	ticker := time.NewTicker(time.Second * miningIntervalSeconds)

	for {
		select {
		case <-ticker.C:
			go func() {
				if len(n.pendingTXs) > 0 && !n.isMining {
					n.isMining = true

					miningCtx, stopCurrentMining = context.WithCancel(ctx)
					err := n.minePendingTXs(miningCtx)
					if err != nil {
						fmt.Printf(rainbow.Red("ERROR: %s\n"), err)
					}

					n.isMining = false
				}
			}()

		case block, _ := <-n.newSyncedBlocks:
			if n.isMining {
				blockHash, _ := block.Hash()
				fmt.Printf("\nPeer mined next Block '%s' faster :(\n", rainbow.Yellow(blockHash.Hex()))

				n.removeMinedPendingTXs(block)
				stopCurrentMining()
			}

		case <-ctx.Done():
			ticker.Stop()
			return nil
		}
	}
}

func (n *Node) minePendingTXs(ctx context.Context) error {
	blockToMine := NewPendingBlock(
		n.state.LatestBlockHash(),
		n.state.NextBlockNumber(),
		n.info.Address,
		n.getPendingTXsAsArray(),
	)
	minedBlock, err := Mine(ctx, blockToMine)
	if err != nil {
		return err
	}

	n.removeMinedPendingTXs(minedBlock)
	_, _, err = n.state.AddBlock(minedBlock)
	if err != nil {
		return err
	}

	return nil
}

func (n *Node) removeMinedPendingTXs(block state.Block) {
	if len(block.TXs) > 0 && len(n.pendingTXs) > 0 {
		fmt.Println("Updating in-memory Pending TXs Pool:")
	}

	for _, tx := range block.TXs {
		txHash, _ := tx.Hash()
		if _, exists := n.pendingTXs[txHash.Hex()]; exists {
			fmt.Printf("\tArchiving mined TX: %s\n", rainbow.Yellow(txHash.Hex()))
			n.archivedTXs[txHash.Hex()] = tx
			delete(n.pendingTXs, txHash.Hex())
		}
	}
}

func (n *Node) AddPeer(peer core.PeerNode) {
	n.knownPeers[peer.TcpAddress()] = peer
}

func (n *Node) RemovePeer(peer core.PeerNode) {
	delete(n.knownPeers, peer.TcpAddress())
}

func (n *Node) IsKnownPeer(peer core.PeerNode) bool {
	if peer.IP == n.info.IP && peer.Port == n.info.Port {
		return true
	}

	_, isKnownPeer := n.knownPeers[peer.TcpAddress()]

	return isKnownPeer
}

/**
Add a pending transaction to the node's pending transactions array
*/
func (n *Node) AddPendingTX(tx state.SignedTx) error {
	txHash, err := tx.Hash()
	if err != nil {
		return err
	}

	_, isAlreadyPending := n.pendingTXs[txHash.Hex()]
	_, isArchived := n.archivedTXs[txHash.Hex()]

	if !isAlreadyPending && !isArchived {
		txJSON, err := json.Marshal(tx)
		if err != nil {
			return err
		}
		prettyTxJSON, err := core.PrettyPrintJSON(txJSON)
		if err != nil {
			return err
		}

		fmt.Printf("Adding pending transactions: \n%s\n", &prettyTxJSON)

		n.pendingTXs[txHash.Hex()] = tx
		n.newPendingTXs <- tx
		tmpFrom := n.state.Manifest[tx.From]
		if tmpFrom.Sent == nil {
			tmpFrom.Inbox = make([]state.InboxItem, 0)
			tmpFrom.Sent = make([]state.SentItem, 0)
			// uncomment the below in order to automatically reward new addresses
			// tmpFrom.Balance += manifest.BlockReward
			// tmpFrom.PendingBalance += tmpFrom.Balance
		}
		// TODO - the cost of the transaction is one coin for now, but should this always be the case?
		//         could file size factor into the cost? -> maybe when I get to the concept of gas?
		if tx.Amount > 0 {
			tmpFrom.PendingBalance -= tx.Amount
		}
		n.state.Manifest[tx.From] = tmpFrom
		// increase the account to nonce value => allows us to support mining blocks with multiple transactions
		n.state.PendingAccount2Nonce[tx.From]++
		tmpTo := n.state.Manifest[tx.To]
		// needed?
		if tmpTo.Inbox == nil {
			tmpTo.Sent = make([]state.SentItem, 0)
			tmpTo.Inbox = make([]state.InboxItem, 0)
			// uncomment the below in order to automatically reward new addresses
			// tmpTo.Balance += manifest.BlockReward
			// tmpTo.PendingBalance += tmpTo.Balance
		}
		tmpTo.PendingBalance += tx.Amount
		n.state.Manifest[tx.To] = tmpTo
	}
	return nil
}

func (n *Node) getPendingTXsAsArray() []state.SignedTx {
	txs := make([]state.SignedTx, len(n.pendingTXs))
	i := 0
	for _, tx := range n.pendingTXs {
		txs[i] = tx
		i++
	}
	return txs
}
