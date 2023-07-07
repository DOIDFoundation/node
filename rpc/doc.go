/*
RPC implementation in DOID node is based on
[github.com/ethereum/go-ethereum/rpc] that follows JSON-RPC 2.0.

# Example

request:

	{"jsonrpc": "2.0", "method": "doid_getBlockByHeight", "params": [1], "id": 1}

response:

	{"jsonrpc":"2.0","id":1,"result":{"header":{"parentHash":"","sha3Uncles":"E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855","miner":"","stateRoot":"E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855","transactionsRoot":"E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855","receiptsRoot":"E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855","difficulty":16777216,"height":1,"timestamp":0,"extraData":"","nonce":"0x0000000000000000"},"data":{"txs":null},"uncles":null,"receipts":null}}

# Request

`method` in request is defined in `{namespace}_{methodName}` format where
  - `namespace` is defined when registering a struct by calling [RegisterName],
    `API List` below shows namespaces and corresponding structs
  - `methodName` is public methods of the struct in uncapitalized form

`params` are the parameters of the public method

# Response

`result` in response is the return values of the public method

# Subscriptions

Subscription is supported in websocket streams.

Method is defined in `{namespace}_subscribe` format. First param is
uncapitalized public method name that returns a
[github.com/ethereum/go-ethereum/rpc.Subscription]

For example:

	{"jsonrpc": "2.0", "method": "doid_subscribe", "params": ["newHeads"], "id": 1}

will execute [github.com/DOIDFoundation/node/core.API.NewHeads]

Result will be like:

	{"jsonrpc":"2.0","id":1,"result":"0xd37be67a9aaa143969d24cada292a445"}
	{"jsonrpc":"2.0","method":"doid_subscription","params":{"subscription":"0xd37be67a9aaa143969d24cada292a445","result":{"header":{"parentHash":"000000A891B5201E56D50B52D619B442E6E89C6AC66ADAF919F3BAFF805924C5","sha3Uncles":"E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855","miner":"","stateRoot":"E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855","transactionsRoot":"E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855","receiptsRoot":"E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855","difficulty":2157515,"height":5286,"timestamp":1688641961,"extraData":"","nonce":"0x45b97c08b2d57274"},"data":{"txs":[]},"uncles":[],"receipts":[]}}}

# API List

Here are some namespaces and their corresponding structs, methods can be found
from the public methods of each struct.

`doid`
  - doid apis [github.com/DOIDFoundation/node/doid.API]
  - blockchain apis [github.com/DOIDFoundation/node/core.API]
  - blockchain subscriptions [github.com/DOIDFoundation/node/core.SubAPI]

`node`
  - [github.com/DOIDFoundation/node/node.API]

`network`
  - [github.com/DOIDFoundation/node/network.API]

`txpool`
  - [github.com/DOIDFoundation/node/mempool.API]
*/
package rpc
