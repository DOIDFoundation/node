# DOID Network Node

## Start

```
go run ./cmd/doidnode start --home /tmp/doid-node-home
```

## Test

```
# list rpc modules
curl -v -s 'localhost:26657' -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","id":1,"method":"rpc_modules"}'

# get node status
curl -v -s 'localhost:26657' -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","id":1,"method":"node_status"}'

# send transaction
curl -v -s 'localhost:26657' -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","id":1,"method":"doid_sendTransaction","params":[{"DOID":"test","Owner":"123421"}]}'
```
