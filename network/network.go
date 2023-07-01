package network

import (
	"context"
	"math/big"
	"sync"
	"sync/atomic"

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
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/viper"
)

// ------------------------------------------------------------------------------
var peerPool = make(map[string]peer.AddrInfo)
var ctx = context.Background()
var peerNotifier = make(chan peer.AddrInfo)

type Network struct {
	service.BaseService
	config *Config

	host          host.Host
	discovery     *discovery
	pubsub        *pubsub.PubSub
	topicBlock    *pubsub.Topic
	blockChain    *core.BlockChain
	networkHeight *big.Int
	sycing        atomic.Bool
	peerPool      map[string]peer.AddrInfo
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

	var opts []libp2p.Option
	addr, err := multiaddr.NewMultiaddr(viper.GetString(flags.P2P_Addr))
	if err != nil {
		network.Logger.Error("Failed to parse p2p.addr", "err", err, "addr", viper.GetString(flags.P2P_Addr))
		return nil
	}
	opts = append(opts, libp2p.ListenAddrs(addr))

	network.host, err = libp2p.New(opts...)
	if err != nil {
		network.Logger.Error("Failed to create libp2p host", "err", err)
		return nil
	}
	network.Logger.Info("Host created.", "id", network.host.ID(), "addrs", network.host.Addrs())

	network.discovery = NewDiscovery(network.host, logger)
	if network.discovery == nil {
		network.Logger.Error("Failed to create discovery service")
		return nil
	}

	network.pubsub, err = pubsub.NewGossipSub(ctx, network.host)
	if err != nil {
		network.Logger.Error("Failed to create pubsub", "err", err)
		return nil
	}

	network.registerEventHandlers()
	return network
}

// OnStart starts the Network. It implements service.Service.
func (n *Network) OnStart() error {
	localHost := n.host

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

	n.discovery.Start()

	go n.Bootstrap(localHost, kademliaDHT)

	return nil
}

// OnStop stops the Network. It implements service.Service.
func (n *Network) OnStop() {
	n.discovery.Stop()
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
