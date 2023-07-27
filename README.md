# DOID Network Node

## Quick Start

- Make a folder for DOID node
- Download and extract available executable files from [releases](https://github.com/DOIDFoundation/node/releases) page
- Download sample config file from [releases](https://github.com/DOIDFoundation/node/releases) page
- Put them in the folder just made
- Execute the doidnode binary

## Start From code

```
git clone https://github.com/DOIDFoundation/node
cd node
go run ./cmd/doidnode start --home /tmp/doid-node-home
```

### Flags

```
go run ./cmd/doidnode start --help
```

Current available flags:

```
Flags:
      --db.engine string         Backing database implementation to use ('memdb' or 'goleveldb') (default "goleveldb")
  -h, --help                     help for start
  -m, --mine.enabled             Enable mining
      --mine.miner string        Miner address to be included in mined blocks
      --mine.threads uint        Number of threads to start mining, 0 indicates number of logical CPUs
      --p2p.addr string          Libp2p listen address (default "/ip4/127.0.0.1/tcp/26667")
      --p2p.key string           Private key to generate libp2p peer identity
      --p2p.keyfile string       Private key file to generate libp2p peer identity (default "p2p.key")
  -r, --p2p.rendezvous string    Libp2p rendezvous string used for peer discovery (default "doidnode")
      --rpc.http.addr string     RPC over HTTP listen address (default "127.0.0.1:8556")
      --rpc.http.enabled         Enable RPC over http
      --rpc.ws.addr string       RPC over websocket listen address (default "127.0.0.1:8557")
      --rpc.ws.enabled           Enable RPC over websocket
      --rpc.ws.origins strings   Origins from which to accept websockets requests (default [*])
```

## Test

```
# list rpc modules
curl -v -s 'localhost:26657' -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","id":1,"method":"rpc_modules"}'

# get node status
curl -v -s 'localhost:26657' -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","id":1,"method":"node_status"}'
```

### test private key

```
prv : ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
address : f39Fd6e51aad88F6F4ce6aB8827279cffFb92266
```

### send transaction

request

````
curl -v -s 'localhost:8556' -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","id":1,"method":"doid_sendTransaction","params":[{"type":"register","data":{"DOID":"test","Owner":"f39Fd6e51aad88F6F4ce6aB8827279cffFb92266", "Signature": "506f3bd07f7015be3495861d5548bca597f3a35ff81df122d0b08ebbbb2aefa52f9aa5bd3b38824d18cd8cce73c35a88518222d7f75f0b6360039f72081701ab01", "From": "f39Fd6e51aad88F6F4ce6aB8827279cffFb92266"}}]}'
````

response

```
{"jsonrpc":"2.0","id":1,"result":"D6392B9662608F2534E12C37BD5D679B2171DF271051E902BAF89E12E0A45512"}
```

### sign

```
sig = crypto.sign(bytes(doidname) + bytes(owner), privatekey)
```

request

```
curl -s 'localhost:8556' -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","id":1,"method":"doid_sign","params":[{"DOID":"test","Owner":"f39Fd6e51aad88F6F4ce6aB8827279cffFb92266","From":"f39Fd6e51aad88F6F4ce6aB8827279cffFb92266", "private":"ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"}]}'
```

response

```
{"jsonrpc":"2.0","id":1,"result":"506f3bd07f7015be3495861d5548bca597f3a35ff81df122d0b08ebbbb2aefa52f9aa5bd3b38824d18cd8cce73c35a88518222d7f75f0b6360039f72081701ab01"}
```

### get owner by doidname

request

````
curl -s 'http://127.0.0.1:8556' -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","id":1,"method":"doid_getOwner","params":[{"DOID":"test"}]}'
```
response
```
{"jsonrpc":"2.0","id":1,"result":"f39fd6e51aad88f6f4ce6ab8827279cfffb92266"}
```
````

### get transaction by hash

request

```
curl -s 'http://127.0.0.1:8556' -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","id":1,"method":"doid_getTransactionByHash","params":[ "D6392B9662608F2534E12C37BD5D679B2171DF271051E902BAF89E12E0A45512"]}'
```

response

````
{"jsonrpc":"2.0","id":1,"result":{"DOID":"test","owner":"F39FD6E51AAD88F6F4CE6AB8827279CFFFB92266","from":"F39FD6E51AAD88F6F4CE6AB8827279CFFFB92266","nameHash":"9C22FF5F21F0B81B113E63F7DB6DA94FEDEF11B2119B4088B89664FB9A3CB658","signature":"","Type":0,"Hash":"D6392B9662608F2534E12C37BD5D679B2171DF271051E902BAF89E12E0A45512"}}```
````
