# DOID Network Node

## Start

```
go run ./cmd/doidnode init --home /tmp/doid-node-home
go run ./cmd/doidnode start --home /tmp/doid-node-home
```

## Test

```
curl -s 'localhost:26657/broadcast_tx_commit?tx="name=abc,owner=0x123,signature=0x123123,code=123"'
curl -s 'localhost:26657/abci_query?data="doid"'
```
