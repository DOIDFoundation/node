package network

import (
	"context"
)

const (
	BlockSyncService              = "BlockSyncRPCAPI"
	EchoServiceFuncEcho           = "Echo"
	BlockSyncServiceFuncGetHeight = "GetBlockHeight"
	BlockSyncServiceFuncGetBlock  = "GetBlock"
)

type BlockSyncRPCAPI struct {
	service *RpcService
}

type Envelope struct {
	Message string
}

type BlockHeightEnvelope struct {
	height int64
}

type BlockEnvelope struct {
	Data []byte
}

func (b *BlockSyncRPCAPI) Echo(ctx context.Context, in Envelope, out *Envelope) error {
	*out = b.service.ReceiveEcho(in)
	return nil
}

func (b *BlockSyncRPCAPI) GetBlockHeight(ctx context.Context, in BlockHeightEnvelope, out *BlockHeightEnvelope) error {
	*out = b.service.ReceiveGetBlockHeight(in)
	return nil
}

func (b *BlockSyncRPCAPI) GetBlock(ctx context.Context, in BlockHeightEnvelope, out *BlockEnvelope) error {
	*out = b.service.ReceiveGetBlock(in)
	return nil
}
