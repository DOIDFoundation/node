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
	taskCh   chan struct{}
	resultCh chan *types.Block
	startCh  chan struct{}
	exitCh   chan struct{}
	chain    *core.BlockChain
}

func New(chain *core.BlockChain, logger log.Logger) *Consensus {
	consensus := &Consensus{
		taskCh:   make(chan struct{}),
		resultCh: make(chan *types.Block),
		startCh:  make(chan struct{}),
		exitCh:   make(chan struct{}),
		chain:    chain,
	}
	consensus.BaseService = *service.NewBaseService(logger.With("module", "consensus"), "Consensus", consensus)
	return consensus
}

func (c *Consensus) OnStart() error {
	c.wg.Add(3)
	go c.mainLoop()
	go c.resultLoop()
	go c.newWorkLoop()
	c.start()
	return nil
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
		case <-c.exitCh:
			return
		}
	}
}

// start a multi thread mining, threads = num of logical CPUs
func (c *Consensus) startMine(stop chan struct{}) error {
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
		case <-c.exitCh:
			// Outside abort, stop all miner threads
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
		parent    = c.chain.CurrentBlock()
		target    = new(big.Int).Div(two256, parent.Header.Difficulty)
		attempts  = int64(0)
		nonce     = seed
		powBuffer = new(big.Int)
		logger    = c.Logger.With("miner", id)
	)

	block := types.NewBlockWithHeader(parent.Header)
	header := block.Header
	header.ParentHash = parent.Hash()
	header.Time = time.Now()
	header.Height.Add(header.Height, big.NewInt(1))
	header.Difficulty.Set(calcDifficulty(header.Time, parent.Header))

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
			c.chain.SetHead(block)
			c.CommitWork()

		case <-c.exitCh:
			return
		}
	}
}

// waiting for submitting new works
func (c *Consensus) newWorkLoop() {
	defer c.wg.Done()

	timer := time.NewTimer(0)
	defer timer.Stop()
	<-timer.C // discard the initial tick
	recommit := 1 * time.Second

	for {
		select {
		case <-c.startCh:
			timer.Reset(recommit)
		case <-timer.C:
			c.CommitWork()
			timer.Reset(recommit)
		case <-c.exitCh:
			return
		}
	}
}

func (c *Consensus) start() {
	c.startCh <- struct{}{}
}

func (c *Consensus) CommitWork() {
	select {
	case c.taskCh <- struct{}{}:
	case <-c.exitCh:
		c.Logger.Info("exiting, work not committed")
		return
	}
}

func (c *Consensus) OnStop() {
	close(c.exitCh)
	c.wg.Wait()
}

// Some weird constants to avoid constant memory allocs for them.
var (
	expDiffPeriod = big.NewInt(100000)
	big1          = big.NewInt(1)
	big2          = big.NewInt(2)
	big9          = big.NewInt(9)
	big10         = big.NewInt(10)
	bigMinus99    = big.NewInt(-99)

	DifficultyBoundDivisor = big.NewInt(2048)   // The bound divisor of the difficulty, used in the update calculations.
	GenesisDifficulty      = big.NewInt(131072) // Difficulty of the Genesis block.
	MinimumDifficulty      = big.NewInt(131072) // The minimum that the difficulty may ever be.
	DurationLimit          = big.NewInt(13)     // The decision boundary on the blocktime duration used to determine whether difficulty should go up or not.
)

func calcDifficulty(time time.Time, parent *types.Header) *big.Int {
	// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-2.md
	// algorithm:
	// diff = (parent_diff +
	//         (parent_diff / 2048 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	//        ) + 2^(periodCount - 2)

	// 1 - (block_timestamp - parent_timestamp) // 10
	x := big.NewInt(time.Unix() - parent.Time.Unix())
	x.Div(x, big10)
	x.Sub(big1, x)

	// max(1 - (block_timestamp - parent_timestamp) // 10, -99)
	if x.Cmp(bigMinus99) < 0 {
		x.Set(bigMinus99)
	}
	y := new(big.Int)
	// (parent_diff + parent_diff // 2048 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	y.Div(parent.Difficulty, DifficultyBoundDivisor)
	x.Mul(y, x)
	x.Add(parent.Difficulty, x)

	// minimum difficulty can ever be (before exponential factor)
	if x.Cmp(MinimumDifficulty) < 0 {
		x.Set(MinimumDifficulty)
	}
	// for the exponential factor
	periodCount := new(big.Int).Add(parent.Height, big1)
	periodCount.Div(periodCount, expDiffPeriod)

	// the exponential factor, commonly referred to as "the bomb"
	// diff = diff + 2^(periodCount - 2)
	if periodCount.Cmp(big1) > 0 {
		y.Sub(periodCount, big2)
		y.Exp(big2, y, nil)
		x.Add(x, y)
	}
	return x
}
