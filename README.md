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
curl -v -s 'localhost:26657' -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","id":1,"method":"doid_sendTransaction","params":[{"DOID":"test","Owner":"f39Fd6e51aad88F6F4ce6aB8827279cffFb92266", "Signature": "0x9a9fbf1a568bf9f1132b90e6a517d8269adcc81386fe9e0e84c2116acedd1d483d9f7ea485ff7e975bd0d1808e533b1862411654237543b605635d03b55dc60801"}]}'

# get owner by doidname
curl -s 'localhost:26657' -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","id":1,"method":"doid_getOwner","params":[{"DOID":"test"}]}'
```

### test private key

```
prv : ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
address : f39Fd6e51aad88F6F4ce6aB8827279cffFb92266
```

### sign

```
curl -s 'localhost:26657' -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","id":1,"method":"doid_sign","params":[{"DOID":"test","Owner":"f39Fd6e51aad88F6F4ce6aB8827279cffFb92266", "private":"ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"}]}'
```
