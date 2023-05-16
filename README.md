# DOID Network Node

## Start

```
go run github.com/cometbft/cometbft/cmd/cometbft@v0.37.0 init --home /tmp/doid-node-home
go run ./cmd/doidnode -datadir /tmp/doid-node-home
```

## Test

```
curl -s 'localhost:26657/broadcast_tx_commit?tx="doid=rocks"'
curl -s 'localhost:26657/abci_query?data="doid"'
```