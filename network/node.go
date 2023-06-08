package network

import (
	"context"
	"fmt"
	"github.com/DOIDFoundation/node/consensus"
	"github.com/DOIDFoundation/node/core"
	//"github.com/DOIDFoundation/node/doid"
	"github.com/DOIDFoundation/node/rpc"
	"github.com/DOIDFoundation/node/store"
	cmtdb "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/cli"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/service"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/multiformats/go-multiaddr"
	"time"

	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

// ------------------------------------------------------------------------------
var peerPool = make(map[string]peer.AddrInfo)
var ctx = context.Background()
var send = Send{}

// Node is the highest level interface to a full network.
// It includes all configuration information and running services.
type Node struct {
	service.BaseService
	config *Config
	rpc    *rpc.RPC

	blockStore       *store.BlockStore
	chain            *core.BlockChain
	consensus        *consensus.Consensus
	localHost        host.Host
	routingDiscovery *drouting.RoutingDiscovery
}

var (
	logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout))
)

// Option sets a parameter for the network.
type Option func(*Node)

// NewNode returns a new, ready to go, CometBFT Node.
func NewNode(logger log.Logger, options ...Option) (*Node, error) {
	homeDir := viper.GetString(cli.HomeFlag)
	db, err := cmtdb.NewDB("chaindata", cmtdb.GoLevelDBBackend, filepath.Join(homeDir, "data"))
	if err != nil {
		return nil, err
	}
	blockStore := store.NewBlockStore(db, logger)

	chain, err := core.NewBlockChain(blockStore, logger)
	if err != nil {
		return nil, err
	}

	node := &Node{
		config: &DefaultConfig,
		rpc:    rpc.NewRPC(logger),

		blockStore: blockStore,
		chain:      chain,
		consensus:  consensus.New(chain, logger),
	}
	node.BaseService = *service.NewBaseService(logger, "Node", node)

	for _, option := range options {
		option(node)
	}

	var opts []libp2p.Option
	node.config.ListenAddresses = viper.GetString("listen")
	node.config.RendezvousString = viper.GetString("rendezvous")
	m1, _ := multiaddr.NewMultiaddr(node.config.ListenAddresses)
	opts = append(opts, libp2p.ListenAddrs(m1))

	localHost, err := libp2p.New(opts...)
	if err != nil {
		panic(err)
	}

	logger.Info("Host created. We are:", localHost.ID(), localHost.Addrs())
	//logger.Info(host.Addrs())

	// Set a function as stream handler. This function is called when a peer
	// initiates a connection and starts a stream with this peer.
	localHost.SetStreamHandler(protocol.ID(ProtocolID), handleStream)

	// Start a DHT, for use in peer discovery. We can't just make a new DHT
	// client because we want each peer to maintain its own local copy of the
	// DHT, so that the bootstrapping network of the DHT can go down without
	// inhibiting future peer discovery.
	kademliaDHT, err := dht.New(ctx, localHost)
	if err != nil {
		panic(err)
	}

	// Bootstrap the DHT. In the default configuration, this spawns a Background
	// thread that will refresh the peer table every five minutes.
	logger.Debug("Bootstrapping the DHT")
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}

	//_ = viper.GetString(node.config.BootstrapPeers)
	if len(node.config.BootstrapPeers) == 0 {
		node.config.BootstrapPeers = dht.DefaultBootstrapPeers
	}
	// Let's connect to the bootstrap nodes first. They will tell us about the
	// other nodes in the network.
	var wg sync.WaitGroup
	for _, peerAddr := range node.config.BootstrapPeers {
		peerinfo2, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := localHost.Connect(ctx, *peerinfo2); err != nil {
				logger.Error(err.Error())
			} else {
				logger.Info("Connection established with bootstrap network:", *peerinfo2)
			}
		}()
	}
	wg.Wait()

	// We use a rendezvous point "meet me here" to announce our location.
	// This is like telling your friends to meet you at the Eiffel Tower.
	logger.Info("Announcing ourselves...", node.config.RendezvousString)
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, node.config.RendezvousString)
	logger.Debug("Successfully announced!")

	node.localHost = localHost
	node.routingDiscovery = routingDiscovery

	send.node = node

	go node.findP2PPeer()
	go node.sendInfo()

	//RegisterAPI(node)
	//if err := doid.RegisterAPI(node.chain); err != nil {
	//	db.Close()
	//	return nil, err
	//}

	return node, nil
}

func (n *Node) findP2PPeer() {
	for {
		// Now, look for others who have announced
		// This is like your friend telling you the location to meet you.
		logger.Debug("Searching for other peers...", n.config.RendezvousString)
		peerChan, err := n.routingDiscovery.FindPeers(ctx, n.config.RendezvousString)
		if err != nil {
			panic(err)
		}

		for peerNode := range peerChan {
			if peerNode.ID == n.localHost.ID() {
				continue
			}
			logger.Info("Found peer:", peerNode)
			peerPool[fmt.Sprint(peerNode.ID)] = peerNode
		}
		time.Sleep(time.Second)
	}
}

func (n *Node) sendInfo() {
	for {
		send.SendTestToPeers()
		time.Sleep(time.Second)
	}
}

func handleStream(stream network.Stream) {
	data, err := ioutil.ReadAll(stream)
	if err != nil {
		logger.Error(err.Error())
	}

	cmd, content := splitMessage(data)
	logger.Info("received command ：%s", cmd)
	switch command(cmd) {
	case cMyTest:
		go handleTest(content)
	case cVersion:
		go handleVersion(content)
	case cGetHash:
	case cHashMap:
	case cGetBlock:
	case cBlock:
	case cTransaction:
	case cMyError:
		//go handleMyError(content)
	}
}

func handleVersion(content []byte) {

}

func handleTest(content []byte) {
	e := myerror{}
	e.deserialize(content)
	logger.Info(e.Error)
}

// OnStart starts the Node. It implements service.Service.
func (n *Node) OnStart() error {
	//if err := n.rpc.Start(); err != nil {
	//	return err
	//}
	//if err := n.consensus.Start(); err != nil {
	//	return err
	//}
	return nil
}

// OnStop stops the Node. It implements service.Service.
func (n *Node) OnStop() {
	n.consensus.Stop()
	n.rpc.Stop()
	n.blockStore.Close()
}
