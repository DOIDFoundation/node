package network

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p/core/protocol"
	"math/big"
	"sync"
	"sync/atomic"

	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"

	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/events"
	"github.com/DOIDFoundation/node/flags"
	"github.com/DOIDFoundation/node/types"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/service"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/viper"
)

// ------------------------------------------------------------------------------
var peerPool = make(map[string]peer.AddrInfo)
var ctx = context.Background()
var peerNotifier = make(chan peer.AddrInfo)
var blockGet = make(chan *big.Int)

type Network struct {
	service.BaseService
	config *Config

	h                host.Host
	routingDiscovery *drouting.RoutingDiscovery
	pubsub           *pubsub.PubSub
	topicBlock       *pubsub.Topic
	topicBlockInfo   *pubsub.Topic
	topicBlockGet    *pubsub.Topic
	blockChain       *core.BlockChain
	networkHeight    *big.Int
	sycing           atomic.Bool
	peerPool         map[string]peer.AddrInfo
}

// Option sets a parameter for the network.
type Option func(*Network)

// NewNetwork returns a new, ready to go, CometBFT Node.
func NewNetwork(chain *core.BlockChain, logger log.Logger) *Network {
	network := &Network{
		config:     &DefaultConfig,
		blockChain: chain,

		networkHeight: big.NewInt(0),
		peerPool:      make(map[string]peer.AddrInfo),
	}
	network.BaseService = *service.NewBaseService(logger.With("module", "network"), "Network", network)

	network.registerEventHandlers()
	return network
}

// OnStart starts the Network. It implements service.Service.
func (n *Network) OnStart() error {
	var opts []libp2p.Option
	n.config.ListenAddresses = viper.GetString(flags.P2P_Addr)
	n.config.RendezvousString = viper.GetString(flags.P2P_Rendezvous)
	m1, err := multiaddr.NewMultiaddr(n.config.ListenAddresses)
	if err != nil {
		return err
	}
	opts = append(opts, libp2p.ListenAddrs(m1))

	localHost, err := libp2p.New(opts...)
	if err != nil {
		return err
	}

	localAddr = fmt.Sprintf(n.config.ListenAddresses+"/p2p/%s", localHost.ID().Pretty())
	n.Logger.Info("Host created. We are:", "localAddr", localAddr)

	n.Logger.Info("Host created. We are:", "id", localHost.ID(), "addrs", localHost.Addrs())
	//n.Logger.Info(host.Addrs())

	// Set a function as stream handler. This function is called when a peer
	// initiates a connection and starts a stream with this peer.
	localHost.SetStreamHandler(protocol.ID(ProtocolID), n.handleStream)

	// Start a DHT, for use in peer discovery. We can't just make a new DHT
	// client because we want each peer to maintain its own local copy of the
	// DHT, so that the bootstrapping network of the DHT can go down without
	// inhibiting future peer discovery.
	kademliaDHT, err := dht.New(ctx, localHost)
	if err != nil {
		return err
	}

	// Bootstrap the DHT. In the default configuration, this spawns a Background
	// thread that will refresh the peer table every five minutes.
	n.Logger.Debug("Bootstrapping the DHT")
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		return err
	}

	//_ = viper.GetString(node.config.BootstrapPeers)
	if len(n.config.BootstrapPeers) == 0 {
		n.config.BootstrapPeers = dht.DefaultBootstrapPeers
	}

	ps, err := pubsub.NewGossipSub(ctx, localHost)
	if err != nil {
		n.Logger.Error("Failed to start pubsub", "err", err)
		return err
	}
	n.pubsub = ps
	n.h = localHost

	go n.Bootstrap(localHost, kademliaDHT)

	return nil
}

// OnStop stops the Network. It implements service.Service.
func (n *Network) OnStop() {
	events.NewChainHead.Unsubscribe(n.String())
	events.ForkDetected.Unsubscribe(n.String())
	events.NewMinedBlock.Unsubscribe(n.String())
}

func (n *Network) startSync() {
	if !n.sycing.CompareAndSwap(false, true) {
		// already started
		return
	}
	n.Logger.Info("start syncing")
	events.SyncStarted.Send(struct{}{})
	events.NewChainHead.Subscribe("network_sync", func(block *types.Block) {
		height := block.Header.Height
		if height.Cmp(n.networkHeight) < 0 {
			n.Logger.Debug("sync in progress", "ours", height, "network", n.networkHeight)
		} else {
			n.Logger.Info("sync caught up", "ours", height, "network", n.networkHeight)
			n.stopSync()
		}
	})
	//to do
}

func (n *Network) stopSync() {
	if !n.sycing.CompareAndSwap(true, false) {
		// already stopped
		return
	}
	events.NewChainHead.Unsubscribe("network_sync")
	events.SyncFinished.Send(struct{}{})
}

func (n *Network) registerEventHandlers() {
	events.ForkDetected.Subscribe(n.String(), func(data struct{}) {
		// @todo handle fork event
		n.startSync()
	})
	events.NewMinedBlock.Subscribe(n.String(), func(data *types.Block) {
		if n.topicBlock == nil {
			n.Logger.Info("not broadcasting, new block topic not joined")
			return
		}
		b, err := rlp.EncodeToBytes(data)
		if err != nil {
			n.Logger.Error("failed to encode block for broadcasting", "err", err)
			return
		}
		n.Logger.Debug("topic peers", "peers", n.topicBlock.ListPeers())
		n.topicBlock.Publish(ctx, b)
	})
}

func (n *Network) Bootstrap(newNode host.Host, kademliaDHT *dht.IpfsDHT) {
	// setup local mDNS discovery
	if err := n.setupDiscovery(newNode); err != nil {
		panic(err)
	}
	// Let's connect to the bootstrap nodes first. They will tell us about the
	// other nodes in the network.
	var wg sync.WaitGroup
	for _, peerAddr := range n.config.BootstrapPeers {
		peerinfo2, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := newNode.Connect(ctx, *peerinfo2); err != nil {
				n.Logger.Error("Failed to connect", "peer", *peerinfo2, "err", err)
			} else {
				n.Logger.Info("Connection established with bootstrap network:", "peer", *peerinfo2)
			}
		}()
	}
	wg.Wait()

	//service := NewService(newNode, protocol.ID(ProtocolID), n.blockChain, n.Logger)
	//err := service.SetupRPC()
	//if err != nil {
	//	n.Logger.Error("new RPC service", "err", err)
	//}

	//n.rpcService = service

	go setupDiscover(ctx, newNode, kademliaDHT, n.config.RendezvousString)
	go n.notifyPeerFoundEvent()
	go n.registerBlockSubscribers()
}

// discoveryNotifee gets notified when we find a new peer via mDNS discovery
type discoveryNotifee struct {
	h      host.Host
	Logger log.Logger
}

func (n *Network) notifyPeerFoundEvent() {
	for {
		pi := <-peerNotifier

		n.peerPool[pi.ID.String()] = pi

		gv := version{Height: n.blockChain.LatestBlock().Header.Height.Int64(), AddrFrom: localAddr}
		data := n.jointMessage(cVersion, gv.serialize())

		n.SendMessage(pi, data)
	}
}

// HandlePeerFound connects to peers discovered via mDNS. Once they're connected,
// the PubSub system will automatically start interacting with them if they also
// support PubSub.
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	if pi.ID.String() == n.h.ID().String() {
		return
	}
	n.Logger.Debug("discovered new peer", "peer", pi)
	err := n.h.Connect(context.Background(), pi)
	if err != nil {
		n.Logger.Debug("error connecting to peer", "peer", pi, "err", err)
	}
	n.Logger.Debug("connected to peer", "peer", pi)

	//peers := n.h.Peerstore().Peers()
	//fmt.Printf("peer len: %d %s\n", len(peers), peers)
	peerNotifier <- pi

}

const DiscoveryServiceTag = "doid-network"

// setupDiscovery creates an mDNS discovery service and attaches it to the libp2p Host.
// This lets us automatically discover peers on the same LAN and connect to them.
func (n *Network) setupDiscovery(h host.Host) error {
	// setup mDNS discovery to find local peers
	s := mdns.NewMdnsService(h, n.config.RendezvousString, &discoveryNotifee{h: h, Logger: n.Logger})
	return s.Start()
}
