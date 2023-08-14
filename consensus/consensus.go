package consensus

import (
	crand "crypto/rand"
	"encoding/binary"
	"github.com/ethereum/go-ethereum/common"
	"math"
	"math/big"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/events"
	"github.com/DOIDFoundation/node/flags"
	"github.com/DOIDFoundation/node/mempool"
	"github.com/DOIDFoundation/node/types"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/service"
	"github.com/spf13/viper"
)

// two256 is a big integer representing 2^256
var two256 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))

type Consensus struct {
	service.BaseService
	wg       sync.WaitGroup
	abort    chan struct{}
	miner    types.Address
	taskCh   chan struct{}
	resultCh chan *types.Block
	chain    *core.BlockChain
	txpool   *mempool.Mempool
	current  *types.Block // current working block
	target   *big.Int     // current difficulty target
}

var (
	// staleThreshold is the maximum depth of the acceptable stale block.
	staleThreshold = 7
)

func New(chain *core.BlockChain, txpool *mempool.Mempool, logger log.Logger) *Consensus {
	consensus := &Consensus{
		miner:    types.HexToAddress(viper.GetString(flags.Mine_Miner)),
		taskCh:   make(chan struct{}),
		resultCh: make(chan *types.Block),
		chain:    chain,
		txpool:   txpool,
	}
	consensus.BaseService = *service.NewBaseService(logger.With("module", "consensus"), "Consensus", consensus)
	if len(consensus.miner) == 0 {
		consensus.Logger.Error("miner account not set, set with --mine.miner")
		return nil
	} else {
		consensus.Logger.Info("miner account", "address", consensus.miner)
	}
	consensus.registerEventHandlers()
	return consensus
}

func (c *Consensus) OnStart() error {
	c.abort = make(chan struct{})
	c.wg.Add(3)
	go c.mainLoop()
	go c.resultLoop()
	go c.newWorkLoop()
	return nil
}

func (c *Consensus) OnStop() {
	close(c.abort)
	c.wg.Wait()
}

func (c *Consensus) OnReset() error {
	return nil
}

func (c *Consensus) registerEventHandlers() {
	var mu sync.Mutex
	events.NewChainHead.Subscribe(c.String(), func(data *types.Block) {
		if c.IsRunning() {
			// chain head changed, now commit a new work.
			c.commitWork()
		}
	})
	events.SyncStarted.Subscribe(c.String(), func(data struct{}) {
		// Enter syncing, now stop mining.
		if c.IsRunning() {
			mu.Lock()
			defer mu.Unlock()
			c.Stop()
			c.Reset()
		}
	})
	events.SyncFinished.Subscribe(c.String(), func(data struct{}) {
		// Sync finished, now start mining.
		mu.Lock()
		defer mu.Unlock()
		c.Start()
	})
}

// the main mining loop
func (c *Consensus) mainLoop() {
	defer c.wg.Done()
	var stopCh chan struct{}
	for {
		select {
		case <-c.taskCh:
			if stopCh != nil {
				close(stopCh)
				stopCh = nil
			}
			stopCh = make(chan struct{})
			if err := c.startMine(stopCh); err != nil {
				c.Logger.Error("block sealing failed", "err", err)
			}
		case <-c.abort:
			if stopCh != nil {
				close(stopCh)
				stopCh = nil
			}
			return
		}
	}
}

// start a multi thread mining, threads = num of logical CPUs
func (c *Consensus) startMine(stop chan struct{}) error {
	abort := make(chan struct{})
	found := make(chan *types.Block)
	threads := viper.GetInt(flags.Mine_Threads)
	if threads == 0 {
		threads = runtime.NumCPU()
	}
	seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return err
	}
	rand := rand.New(rand.NewSource(seed.Int64()))
	c.Logger.Info("start a new mine round", "threads", threads)
	var pend sync.WaitGroup
	for i := 0; i < threads; i++ {
		pend.Add(1)
		go func(id int, nonce uint64) {
			defer pend.Done()
			c.mine(id, nonce, abort, found)
		}(i, uint64(rand.Int63()))
	}

	// Wait until sealing is terminated or a nonce is found
	go func() {
		var result *types.Block
		select {
		case <-stop:
			// Outside abort, stop all miner threads
			close(abort)
		case result = <-found:
			// One of the threads found a block, abort all others
			select {
			case c.resultCh <- result:
			default:
				c.Logger.Info("result is not read by miner", "header", result.Header, "hash", result.Hash())
			}
			close(abort)
		}
		// Wait for all miners to terminate
		pend.Wait()
	}()
	return nil
}

// actual mining function
func (c *Consensus) mine(id int, seed uint64, abort chan struct{}, found chan *types.Block) {
	var (
		target    = c.target
		attempts  = int64(0)
		nonce     = seed
		powBuffer = new(big.Int)
		logger    = c.Logger.With("miner", id)
		block     = types.NewBlockWithHeader(c.current.Header)
		header    = block.Header
	)

	block.Txs = c.current.Txs
	block.Uncles = c.current.Uncles
	block.Receipts = c.current.Receipts

	logger.Debug("started search for new nonces", "seed", seed, "target", target.Text(16))

search:
	for {
		select {
		case <-abort:
			logger.Debug("nonce search aborted", "attempts", nonce-seed)
			break search

		default:
			attempts++
			if (attempts % (1 << 15)) == 0 {
				logger.Debug("nonce searching", "attempts", nonce-seed)
				attempts = 0
			}
			// Compute the PoW value of this nonce
			binary.BigEndian.PutUint64(header.Nonce[:], nonce)
			hash := block.Hash()
			if powBuffer.SetBytes(hash).Cmp(target) <= 0 {
				select {
				case found <- block:
					logger.Debug("nonce found and reported", "attempts", nonce-seed, "nonce", nonce, "hash", hash)
				case <-abort:
					logger.Debug("nonce found but discarded", "attempts", nonce-seed, "nonce", nonce)
				}
				break search
			}
			nonce++
		}
	}
}

// waiting for mining results
func (c *Consensus) resultLoop() {
	defer c.wg.Done()
	for {
		select {
		case block := <-c.resultCh:
			if err := c.chain.ApplyBlock(block); err != nil {
				c.Logger.Error("error applying found block", "err", err)
			} else {
				events.NewMinedBlock.Send(block)
			}
			c.commitWork()

		case <-c.abort:
			return
		}
	}
}

// waiting for submitting new works
func (c *Consensus) newWorkLoop() {
	defer c.wg.Done()

	recommit := 1 * time.Second
	timer := time.NewTimer(recommit)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			c.commitWork()
			timer.Reset(recommit)
		case <-c.abort:
			return
		}
	}
}

func (c *Consensus) commitWork() {
	txs := c.txpool.Pending()
	result, err := c.chain.Simulate(txs)
	if err != nil {
		c.Logger.Error("failed to simulate transactions", "err", err)
		return
	}

	var (
		parent = c.chain.LatestBlock()
		block  = types.NewBlockWithHeader(parent.Header)
		header = block.Header
	)

	c.target = new(big.Int).Div(two256, header.Difficulty)

	header.ParentHash = parent.Hash()
	header.Miner = c.miner
	header.Time = uint64(time.Now().Unix())
	header.Height.Add(header.Height, big.NewInt(1))
	header.Difficulty.Set(types.CalcDifficulty(header.Time, parent.Header))
	header.Root = result.StateRoot
	header.TxHash = result.TxRoot
	header.ReceiptHash = result.ReceiptRoot
	block.Txs = result.Txs
	block.Receipts = result.Receipts

	// Accumulate the uncles for the current block
	uncles := make([]*types.Header, 0, 2)
	commitUncles := func(blocks map[common.Hash]*types.Block) {
		// Clean up stale uncle blocks first
		for hash, uncle := range blocks {
			if uncle.Header.Height.Uint64()+uint64(staleThreshold) <= header.Height.Uint64() {
				delete(blocks, hash)
			}
		}
		for hash, uncle := range blocks {
			if len(uncles) == 2 {
				break
			}
			if err := c.chain.CommitUncle(uncle.Header); err != nil {
				c.Logger.Info("Possible uncle rejected", "hash", hash, "reason", err)
			} else {
				c.Logger.Debug("Committing new uncle to block", "hash", hash)
				uncles = append(uncles, uncle.Header)
			}
		}
	}
	// Prefer to locally generated uncle
	commitUncles(c.chain.LocalUncles)

	block.Uncles = uncles
	c.current = block

	select {
	case c.taskCh <- struct{}{}:
	case <-c.abort:
		c.Logger.Info("exiting, work not committed")
		return
	}
}
