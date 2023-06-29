package network

import (
	"context"
	"fmt"
	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/types"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/libp2p/go-libp2p/core/peer"
	"time"

	"github.com/libp2p/go-libp2p-gorpc"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/protocol"
)

type RpcService struct {
	Logger      log.Logger
	rpcServer   *rpc.Server
	rpcClient   *rpc.Client
	host        host.Host
	protocol    protocol.ID
	counter     int
	blockChain  *core.BlockChain
	newStatusCh chan struct{}
}

func NewService(host host.Host, protocol protocol.ID, blockChain *core.BlockChain, Logger log.Logger) *RpcService {
	return &RpcService{
		host:        host,
		protocol:    protocol,
		Logger:      Logger,
		blockChain:  blockChain,
		newStatusCh: make(chan struct{}),
	}
}

func (s *RpcService) SetupRPC() error {
	s.rpcServer = rpc.NewServer(s.host, s.protocol)

	echoRPCAPI := BlockSyncRPCAPI{service: s}
	err := s.rpcServer.Register(&echoRPCAPI)
	if err != nil {
		return err
	}

	s.rpcClient = rpc.NewClientWithServer(s.host, s.protocol, s.rpcServer)
	return nil
}

func (s *RpcService) notifyNewStatusEvent() {
	select {
	case s.newStatusCh <- struct{}{}:
	default:
	}
}

func (s *RpcService) StartMessaging(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.counter++
			s.Echo(fmt.Sprintf("Message (%d): Hello from %s", s.counter, s.host.ID().Pretty()))
		}
	}
}

func (s *RpcService) Echo(message string) {
	peers := s.host.Peerstore().Peers()
	var replies = make([]*Envelope, len(peers))

	errs := s.rpcClient.MultiCall(
		Ctxts(len(peers)),
		peers,
		BlockSyncService,
		EchoServiceFuncEcho,
		Envelope{Message: message},
		CopyEnvelopesToIfaces(replies),
	)

	for i, err := range errs {
		if err != nil {
			fmt.Printf("Peer %s returned error: %-v\n", peers[i].Pretty(), err)
		} else {
			fmt.Printf("Peer %s echoed: %s\n", peers[i].Pretty(), replies[i].Message)
		}
	}
}

func (s *RpcService) StartBlockSync() {
	for {
		// Wait for a new event to arrive
		<-s.newStatusCh
		peers := s.host.Peerstore().Peers()
		//var replies = make([]*BlockHeightEnvelope, len(peers))
		//
		//errs := s.rpcClient.MultiCall(
		//	Ctxts(len(peers)),
		//	peers,
		//	BlockSyncService,
		//	BlockSyncServiceFuncGetHeight,
		//	BlockHeightEnvelope{height: 0},
		//	CopyBlockHeightEnvelopesToIfaces(replies),
		//)

		localHeight := s.blockChain.LatestBlock().Header.Height.Uint64()
		for _, peer := range peers {
			if s.host.ID() == peer {
				continue
			}
			var blockHeightEnvelope BlockHeightEnvelope
			err := s.rpcClient.Call(
				peer,
				BlockSyncService,
				BlockSyncServiceFuncGetHeight,
				BlockHeightEnvelope{height: localHeight},
				&blockHeightEnvelope,
			)

			if err != nil {
				fmt.Printf("Peer %s returned error: %-v\n", peer.Pretty(), err)
			} else {
				fmt.Printf("Peer %s height: %d localHeight: %d\n", peer.Pretty(), blockHeightEnvelope.height, s.blockChain.LatestBlock().Header.Height.Uint64())
			}
			if localHeight < blockHeightEnvelope.height {
				s.BlockSync(peer)
			}
		}
	}

}
func (s *RpcService) BlockSync(peer peer.ID) {
	fmt.Printf("Peer %s begin receive block\n", peer.Pretty())
	for {
		var blockEnvelope BlockEnvelope
		fmt.Printf("Peer %s begin receive block height: %d\n", peer.Pretty(), s.blockChain.LatestBlock().Header.Height.Uint64()+1)
		err := s.rpcClient.Call(
			peer,
			BlockSyncService,
			BlockSyncServiceFuncGetBlock,
			BlockHeightEnvelope{height: s.blockChain.LatestBlock().Header.Height.Uint64() + 1},
			&blockEnvelope,
		)

		fmt.Printf("Peer %s receive block height: %d\n", peer.Pretty(), s.blockChain.LatestBlock().Header.Height.Uint64()+1)
		if err == nil {
			fmt.Printf("Peer %s receive block size: %d\n", peer.Pretty(), len(blockEnvelope.Data))
			block := new(types.Block)
			err := rlp.DecodeBytes(blockEnvelope.Data, block)
			if err != nil {
				s.Logger.Info("failed to decode received block", "Peer", peer.Pretty(), "err", err)
				return
			}
			if s.blockChain.LatestBlock().Header.Height.Int64() < block.Header.Height.Int64() {
				s.Logger.Info("apply block", "Peer", peer.Pretty(), "block height", block.Header.Height)
				err = s.blockChain.ApplyBlock(block)
				if err != nil {
					s.Logger.Info("failed to apply block", "Peer", peer.Pretty(), "err", err)
				}
			} else {
				return
			}
		}
	}
}

func (s *RpcService) ReceiveEcho(envelope Envelope) Envelope {
	return Envelope{Message: fmt.Sprintf("Peer %s echoing: %s", s.host.ID(), envelope.Message)}
}

func (s *RpcService) ReceiveGetBlockHeight(envelope BlockHeightEnvelope) BlockHeightEnvelope {
	fmt.Printf("send height: %d\n", s.blockChain.LatestBlock().Header.Height.Uint64())
	return BlockHeightEnvelope{height: s.blockChain.LatestBlock().Header.Height.Uint64()}
}

func (s *RpcService) ReceiveGetBlock(envelope BlockHeightEnvelope) BlockEnvelope {
	block := s.blockChain.BlockByHeight(envelope.height)

	b, err := rlp.EncodeToBytes(block)
	s.Logger.Info("encode block for broadcasting", "size", len(b), "height", envelope.height)
	if err != nil {
		s.Logger.Info("failed to encode block for broadcasting", "err", err)
		return BlockEnvelope{Data: []byte("")}
	}
	fmt.Printf("send block size: %d\n", len(b))
	return BlockEnvelope{Data: b}
}

func Ctxts(n int) []context.Context {
	ctxs := make([]context.Context, n)
	for i := 0; i < n; i++ {
		ctxs[i] = context.Background()
	}
	return ctxs
}

func CopyEnvelopesToIfaces(in []*Envelope) []interface{} {
	ifaces := make([]interface{}, len(in))
	for i := range in {
		in[i] = &Envelope{}
		ifaces[i] = in[i]
	}
	return ifaces
}

func CopyBlockHeightEnvelopesToIfaces(in []*BlockHeightEnvelope) []interface{} {
	ifaces := make([]interface{}, len(in))
	for i := range in {
		in[i] = &BlockHeightEnvelope{}
		ifaces[i] = in[i]
	}
	return ifaces
}
