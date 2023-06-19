package network

import (
	"context"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/flags"
	"github.com/DOIDFoundation/node/types"
	"github.com/cometbft/cometbft/libs/events"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/service"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/viper"
)

// ------------------------------------------------------------------------------
var peerPool = make(map[string]peer.AddrInfo)
var ctx = context.Background()
var send = Send{}

type Network struct {
	service.BaseService
	config *Config

	localHost        host.Host
	routingDiscovery *drouting.RoutingDiscovery
	pubsub           *pubsub.PubSub
	topicBlock       *pubsub.Topic
}

// Option sets a parameter for the network.
type Option func(*Network)

// NewNetwork returns a new, ready to go, CometBFT Node.
func NewNetwork(logger log.Logger) *Network {
	network := &Network{
		config: &DefaultConfig,
	}
	network.BaseService = *service.NewBaseService(logger.With("module", "network"), "Network", network)

	return network
}

// OnStart starts the Network. It implements service.Service.
func (n *Network) OnStart() error {
	var opts []libp2p.Option
	n.config.ListenAddresses = viper.GetString(flags.P2P_Addr)
	n.config.RendezvousString = viper.GetString("rendezvous")
	m1, err := multiaddr.NewMultiaddr(n.config.ListenAddresses)
	if err != nil {
		return err
	}
	opts = append(opts, libp2p.ListenAddrs(m1))

	localHost, err := libp2p.New(opts...)
	if err != nil {
		return err
	}

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

	gs, err := pubsub.NewGossipSub(ctx, localHost)
	if err != nil {
		n.Logger.Error("Failed to start pubsub", "err", err)
		return err
	}
	n.pubsub = gs
	n.registerEventHandlers()
	n.registerSubscribers()

	go n.Bootstrap(localHost, kademliaDHT)
	return nil
}

func (n *Network) registerEventHandlers() {
	core.EventInstance().AddListenerForEvent(n.String(), types.EventNewBlock, func(data events.EventData) {
		b, err := rlp.EncodeToBytes(data.(*types.Block))
		if err != nil {
			n.Logger.Error("failed to encode block for broadcasting", "err", err)
			return
		}
		n.Logger.Debug("topic peers", "peers", n.topicBlock.ListPeers())
		n.topicBlock.Publish(ctx, b)
	})
}

func (n *Network) registerSubscribers() {
	// @todo use different topic for different fork
	topic, err := n.pubsub.Join("/doid/block")
	if err != nil {
		n.Logger.Error("Failed to join pubsub topic", "err", err)
		return
	}
	sub, err := topic.Subscribe()
	if err != nil {
		topic.Close()
		n.Logger.Error("Failed to subscribe to pubsub topic", "err", err)
		return
	}
	n.topicBlock = topic

	// Pipeline decodes the incoming subscription data, runs the validation, and handles the
	// message.
	pipeline := func(msg *pubsub.Message) {
		data := msg.GetData()
		block := new(types.Block)
		err := rlp.DecodeBytes(data, block)
		if err != nil {
			n.Logger.Error("failed to decode received block", "err", err)
			return
		}
		n.Logger.Debug("got message", "block", *block)
		// @todo handle block message
	}

	// The main message loop for receiving incoming messages from this subscription.
	messageLoop := func() {
		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				// This should only happen when the context is cancelled or subscription is cancelled.
				if err != pubsub.ErrSubscriptionCancelled { // Only log an error on unexpected errors.
					n.Logger.Error("Subscription next failed", "err", err)
				}
				// Cancel subscription in the event of an error, as we are
				// now exiting topic event loop.
				sub.Cancel()
				return
			}

			if msg.ReceivedFrom == n.localHost.ID() {
				continue
			}

			go pipeline(msg)
		}
	}

	go messageLoop()
}

func (n *Network) Bootstrap(localHost host.Host, kademliaDHT *dht.IpfsDHT) {
	// Let's connect to the bootstrap nodes first. They will tell us about the
	// other nodes in the network.
	var wg sync.WaitGroup
	for _, peerAddr := range n.config.BootstrapPeers {
		peerinfo2, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := localHost.Connect(ctx, *peerinfo2); err != nil {
				n.Logger.Error("Failed to connect", "peer", *peerinfo2, "err", err)
			} else {
				n.Logger.Info("Connection established with bootstrap network:", "peer", *peerinfo2)
			}
		}()
	}
	wg.Wait()

	// We use a rendezvous point "meet me here" to announce our location.
	// This is like telling your friends to meet you at the Eiffel Tower.
	n.Logger.Info("Announcing ourselves...", n.config.RendezvousString)
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, n.config.RendezvousString)
	n.Logger.Debug("Successfully announced!")

	n.localHost = localHost
	n.routingDiscovery = routingDiscovery

	send.node = n

	go n.findP2PPeer()
	go n.sendInfo()
}

func (n *Network) findP2PPeer() {
	for {
		// Now, look for others who have announced
		// This is like your friend telling you the location to meet you.
		n.Logger.Debug("Searching for other peers...", n.config.RendezvousString)
		peerChan, err := n.routingDiscovery.FindPeers(ctx, n.config.RendezvousString)
		if err != nil {
			panic(err)
		}

		for peerNode := range peerChan {
			if peerNode.ID == n.localHost.ID() {
				continue
			}
			n.Logger.Info("Found peer:", peerNode)
			peerPool[fmt.Sprint(peerNode.ID)] = peerNode
		}
		time.Sleep(time.Second)
	}
}

func (n *Network) sendInfo() {
	for {
		send.SendTestToPeers()
		time.Sleep(time.Second)
	}
}

func (n *Network) handleStream(stream network.Stream) {
	data, err := ioutil.ReadAll(stream)
	if err != nil {
		n.Logger.Error(err.Error())
	}

	cmd, content := splitMessage(data)
	n.Logger.Info("received command ：%s", cmd)
	switch command(cmd) {
	case cMyTest:
		go n.handleTest(content)
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

func (n *Network) handleTest(content []byte) {
	e := myerror{}
	if err := e.deserialize(content); err != nil {
		n.Logger.Error("failed to deserialize content", "err", err)
	}
	n.Logger.Info(e.Error)
}

// OnStop stops the Network. It implements service.Service.
func (n *Network) OnStop() {
}
