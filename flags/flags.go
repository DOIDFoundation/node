package flags

const (
	Home  = "home"
	Trace = "trace"

	Mainnet   = "mainnet"
	Testnet   = "testnet"
	NetworkID = "networkid"

	DB_Engine = "db.engine"

	Log_Level = "log.level"

	Mine_Enabled = "mine.enabled"
	Mine_Threads = "mine.threads"
	Mine_Miner   = "mine.miner"

	P2P_Addr       = "p2p.addr"
	P2P_Rendezvous = "p2p.rendezvous"
	P2P_Key        = "p2p.key"
	P2P_KeyFile    = "p2p.keyfile"

	RPC_Http      = "rpc.http.enabled"
	RPC_HttpAddr  = "rpc.http.addr"
	RPC_Ws        = "rpc.ws.enabled"
	RPC_WsAddr    = "rpc.ws.addr"
	RPC_WsOrigins = "rpc.ws.origins"
)
