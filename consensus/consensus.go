package consensus

import (
	crand "crypto/rand"
	"encoding/binary"
	"math"
	"math/big"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/types"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/service"
)

// two256 is a big integer representing 2^256
var two256 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))

type Consensus struct {
	service.BaseService
	wg       sync.WaitGroup
	taskCh   chan *types.Block
	resultCh chan *types.Block
	exitCh   chan struct{}
	chain    *core.BlockChain
}

func New(chain *core.BlockChain, logger log.Logger) *Consensus {
	consensus := &Consensus{
		taskCh:   make(chan *types.Block),
		resultCh: make(chan *types.Block),
		exitCh:   make(chan struct{}),
		chain:    chain,
	}
	consensus.BaseService = *service.NewBaseService(logger.With("module", "consensus"), "Consensus", consensus)
	return consensus
}

func (c *Consensus) OnStart() error {
	c.wg.Add(2)
	go c.mainLoop()
	go c.resultLoop()
	return nil
}

func (c *Consensus) mainLoop() {
	defer c.wg.Done()
	for {
		select {
		case task := <-c.taskCh:
			if err := c.startMine(task); err != nil {
				c.Logger.Error("block sealing failed", "err", err)
			}
		case <-c.exitCh:
			return
		}
	}
}

func (c *Consensus) startMine(task *types.Block) error {
	abort := make(chan struct{})
	found := make(chan *types.Block)
	threads := runtime.NumCPU()
	seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return err
	}
	rand := rand.New(rand.NewSource(seed.Int64()))
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
		case result = <-found:
			// One of the threads found a block, abort all others
			select {
			case c.resultCh <- result:
			default:
				c.Logger.Info("result is not read by miner", "header", result.Header, "hash", result.Hash())
			}
			close(abort)
		case <-c.exitCh:
			// Outside abort, stop all miner threads
			close(abort)
		}
		// Wait for all miners to terminate
		pend.Wait()
	}()
	return nil
}

func (c *Consensus) mine(id int, seed uint64, abort chan struct{}, found chan *types.Block) {
	var (
		header    = c.chain.CurrentBlock().Header
		block     = types.NewBlockWithHeader(header)
		target    = new(big.Int).Div(two256, header.Difficulty)
		attempts  = int64(0)
		nonce     = seed
		powBuffer = new(big.Int)
		logger    = c.Logger.With("miner", id)
	)

	header = block.Header
	header.Height.Add(header.Height, big.NewInt(1))
	header.ParentHash = block.Hash()
	header.Time = time.Now()

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

func (c *Consensus) resultLoop() {
	defer c.wg.Done()
	for {
		select {
		case block := <-c.resultCh:
			c.chain.SetHead(block)
			time.Sleep(time.Second)
			c.CommitWork(block)

		case <-c.exitCh:
			return
		}
	}
}

func (c *Consensus) CommitWork(block *types.Block) {
	select {
	case c.taskCh <- block:
	case <-c.exitCh:
		c.Logger.Info("exiting, work not committed")
		return
	}
}

func (c *Consensus) OnStop() {
	close(c.exitCh)
	c.wg.Wait()
}
