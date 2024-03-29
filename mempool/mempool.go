package mempool

import (
	"bytes"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/DOIDFoundation/node/config"
	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/events"
	"github.com/DOIDFoundation/node/types"
	"github.com/DOIDFoundation/node/types/tx"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/service"
)

var (
	evictionInterval = time.Minute // Time interval to check for evictable transactions
)

var (
	ErrAlreadyKnown   = errors.New("already known")
	ErrTxPoolOverflow = errors.New("txpool is full")
)

type Mempool struct {
	service.BaseService
	chain *core.BlockChain
	mu    sync.RWMutex

	// pending map[common.Address]*txList   // All currently processable transactions
	// queue   map[common.Address]*txList   // Queued but non-processable transactions
	all *txLookup // All transactions to allow lookups

	reqResetCh      chan *txpoolResetRequest
	queueTxEventCh  chan types.Tx
	reorgShutdownCh chan struct{} // requests shutdown of scheduleReorgLoop
	reorgDoneCh     chan chan struct{}
	wg              sync.WaitGroup // tracks loop, scheduleReorgLoop
}

type txpoolResetRequest struct {
	oldHead, newHead *types.Header
}

func NewMempool(chain *core.BlockChain, logger log.Logger) *Mempool {
	pool := &Mempool{
		chain:           chain,
		all:             newTxLookup(),
		queueTxEventCh:  make(chan types.Tx),
		reorgShutdownCh: make(chan struct{}),
		reorgDoneCh:     make(chan chan struct{}),
		reqResetCh:      make(chan *txpoolResetRequest),
	}
	pool.BaseService = *service.NewBaseService(logger.With("module", "mempool"), "mempool", pool)
	return pool
}

func (pool *Mempool) OnStart() error {
	pool.registerEventHandlers()

	pool.wg.Add(1)
	go pool.scheduleReorgLoop()

	pool.wg.Add(1)
	go pool.loop()
	return nil
}

func (pool *Mempool) registerEventHandlers() {
	header := pool.chain.LatestBlock().Header
	pool.reset(nil, header)
	events.NewChainHead.Subscribe(pool.String(), func(block *types.Block) {
		pool.requestReset(header, block.Header)
		header = block.Header
	})
	onTx := func(data types.Tx) {
		if networkId := tx.ChainId(data); networkId != config.NetworkID {
			pool.Logger.Debug("discard transaction, chain id mismatch", "networkId", networkId)
			return
		}
		err := pool.AddLocal(data)
		if err != nil {
			pool.Logger.Error("failed to add transaction", "err", err)
		}
	}
	events.NewTx.Subscribe(pool.String(), onTx)
	events.NewNetworkTx.Subscribe(pool.String(), onTx)
}

func (pool *Mempool) OnStop() {
	events.NewChainHead.Unsubscribe(pool.String())
	events.NewTx.Unsubscribe(pool.String())
	events.NewNetworkTx.Unsubscribe(pool.String())
}

func (pool *Mempool) Stats() (int, int) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.all.LocalCount(), pool.all.RemoteCount()
}

func (pool *Mempool) loop() {
	defer pool.wg.Done()

	var (
		evict = time.NewTicker(evictionInterval)
	)

	for {
		select {
		case <-evict.C:
			pool.mu.Lock()
			// for addr := range pool.queue {
			// 	// Skip local transactions from the eviction mechanism
			// 	if pool.locals.contains(addr) {
			// 		continue
			// 	}
			// 	// Any non-locals old enough should be removed
			// 	if time.Since(pool.beats[addr]) > pool.config.Lifetime {
			// 		list := pool.queue[addr].Flatten()
			// 		for _, tx := range list {
			// 			pool.removeTx(tx.Hash(), true)
			// 		}
			// 		queuedEvictionMeter.Mark(int64(len(list)))
			// 	}
			// }
			pool.mu.Unlock()
		}
	}
}

// requestReset requests a pool reset to the new head block.
// The returned channel is closed when the reset has occurred.
func (pool *Mempool) requestReset(oldHead *types.Header, newHead *types.Header) chan struct{} {
	select {
	case pool.reqResetCh <- &txpoolResetRequest{oldHead, newHead}:
		return <-pool.reorgDoneCh
	case <-pool.reorgShutdownCh:
		return pool.reorgShutdownCh
	}
}

// queueTxEvent enqueues a transaction event to be sent in the next reorg run.
func (pool *Mempool) queueTxEvent(tx types.Tx) {
	select {
	case pool.queueTxEventCh <- tx:
	case <-pool.reorgShutdownCh:
	}
}

func (pool *Mempool) validateTx(tx types.Tx) (err error) {
	return nil
}

func (pool *Mempool) AddLocals(txs []types.Tx) []error {
	return pool.AddTxs(txs)
}

func (pool *Mempool) AddLocal(tx types.Tx) error {
	return pool.AddLocals(types.Txs{tx})[0]
}

func (pool *Mempool) add(tx types.Tx, local bool) (replaced bool, err error) {
	hash := tx.Key()
	if pool.all.Get(hash) != nil {
		return false, ErrAlreadyKnown
	}
	if err := pool.validateTx(tx); err != nil {
		return false, err
	}

	// check pool is full
	if pool.all.Slots()+numSlots(tx) > 100 {
		return false, ErrTxPoolOverflow
	}
	// pool.queueTxEvent(tx)

	err = pool.enqueueTx(hash, tx)
	if err != nil {
		return false, err
	}

	return false, nil
}

func (pool *Mempool) enqueueTx(hash types.TxHash, tx types.Tx) error {
	pool.all.Add(tx, true)
	return nil
}

func (pool *Mempool) AddTxs(txs []types.Tx) []error {
	var (
		errs = make([]error, len(txs))
		news = make(types.Txs, 0, len(txs))
	)
	for i, tx := range txs {
		// If the transaction is known, pre-set the error slot
		if pool.all.Get(tx.Key()) != nil {
			errs[i] = ErrAlreadyKnown
			// knownTxMeter.Mark(1)
			continue
		}
		news = append(news, tx)
	}
	if len(news) == 0 {
		return errs
	}

	// Process all the new transaction and merge any errors into the original slice
	pool.mu.Lock()
	newErrs := make([]error, len(txs))
	for i, tx := range txs {
		_, err := pool.add(tx, true)
		newErrs[i] = err
	}
	pool.mu.Unlock()

	var nilSlot = 0
	for _, err := range newErrs {
		for errs[nilSlot] != nil {
			nilSlot++
		}
		errs[nilSlot] = err
		nilSlot++
	}
	return errs
}

// remove removes a single transaction from the queue.
func (pool *Mempool) remove(hash types.TxHash) {
	// Fetch the transaction we wish to delete
	tx := pool.all.Get(hash)
	if tx == nil {
		return
	}

	// Remove it from the list of known transactions
	pool.all.Remove(hash)
}

func (pool *Mempool) Pending() types.Txs {
	return pool.all.Flatten()
}

func (pool *Mempool) scheduleReorgLoop() {
	defer pool.wg.Done()

	var (
		curDone       chan struct{} // non-nil while runReorg is active
		nextDone      = make(chan struct{})
		launchNextRun bool
		reset         *txpoolResetRequest
		queuedTx      types.Tx
	)

	for {
		if curDone == nil && launchNextRun {
			go pool.runReorg(nextDone, reset, queuedTx)
			curDone, nextDone = nextDone, make(chan struct{})
			launchNextRun = false
			queuedTx = nil
			reset = nil
		}

		select {
		case req := <-pool.reqResetCh:
			// Reset request: update head if request is already pending.
			if reset == nil {
				reset = req
			} else {
				reset.newHead = req.newHead
			}
			launchNextRun = true
			pool.reorgDoneCh <- nextDone

		case tx := <-pool.queueTxEventCh:
			launchNextRun = true
			queuedTx = tx

		case <-curDone:
			curDone = nil
		}

	}
}

func (pool *Mempool) runReorg(done chan struct{}, reset *txpoolResetRequest, tx types.Tx) {
	defer close(done)

	pool.mu.Lock()
	if reset != nil {
		// Reset from the old head to the new, rescheduling any reorged transactions
		pool.reset(reset.oldHead, reset.newHead)

	}
	// pool.truncatePending()
	// pool.truncateQueue()
	pool.mu.Unlock()
}

func (pool *Mempool) reset(oldHead, newHead *types.Header) {
	// Remove all accepted transactions, reinject all dropped transactions if we're reorging an old state.
	var reinject, accepted types.Txs

	if oldHead == nil {
		// start from genesis
		oldHead = pool.chain.BlockByHeight(1).Header
	}
	if bytes.Equal(oldHead.Hash(), newHead.ParentHash) {
		accepted = pool.chain.GetBlock(newHead.Height.Uint64(), newHead.Hash()).Txs
	} else {
		// If the reorg is too deep, avoid doing it (will happen during fast sync)
		oldNum := oldHead.Height.Uint64()
		newNum := newHead.Height.Uint64()

		if depth := uint64(math.Abs(float64(oldNum) - float64(newNum))); depth > 64 {
			pool.Logger.Debug("Skipping deep transaction reorg", "depth", depth)
		} else {
			// Reorg seems shallow enough to pull in all transactions into memory
			var discarded, included types.Txs
			var (
				rem = pool.chain.GetBlock(oldHead.Height.Uint64(), oldHead.Hash())
				add = pool.chain.GetBlock(newHead.Height.Uint64(), newHead.Hash())
			)
			if rem == nil {
				// This can happen if a setHead is performed, where we simply discard the old
				// head from the chain.
				// If that is the case, we don't have the lost transactions any more, and
				// there's nothing to add
				if newNum < oldNum {
					// If the reorg ended up on a lower number, it's indicative of setHead being the cause
					pool.Logger.Debug("Skipping transaction reset caused by setHead",
						"old", oldHead.Hash(), "oldnum", oldNum, "new", newHead.Hash(), "newnum", newNum)
				} else {
					// If we reorged to a same or higher number, then it's not a case of setHead
					pool.Logger.Error("Transaction pool reset with missing oldhead",
						"old", oldHead.Hash(), "oldnum", oldNum, "new", newHead.Hash(), "newnum", newNum)
				}
				return
			}
			for rem.Header.Height.Cmp(add.Header.Height) > 0 {
				discarded = append(discarded, rem.Txs...)
				if rem = pool.chain.GetBlock(rem.Header.Height.Uint64()-1, rem.Header.ParentHash); rem == nil {
					pool.Logger.Error("Unrooted old chain seen by tx pool", "block", oldHead.Height, "hash", oldHead.Hash())
					return
				}
			}
			for add.Header.Height.Cmp(rem.Header.Height) > 0 {
				included = append(included, add.Txs...)
				if add = pool.chain.GetBlock(add.Header.Height.Uint64()-1, add.Header.ParentHash); add == nil {
					pool.Logger.Error("Unrooted new chain seen by tx pool", "block", newHead.Height, "hash", newHead.Hash())
					return
				}
			}
			for !bytes.Equal(rem.Hash(), add.Hash()) {
				discarded = append(discarded, rem.Txs...)
				if rem = pool.chain.GetBlock(rem.Header.Height.Uint64()-1, rem.Header.ParentHash); rem == nil {
					pool.Logger.Error("Unrooted old chain seen by tx pool", "block", oldHead.Height, "hash", oldHead.Hash())
					return
				}
				included = append(included, add.Txs...)
				if add = pool.chain.GetBlock(add.Header.Height.Uint64()-1, add.Header.ParentHash); add == nil {
					pool.Logger.Error("Unrooted new chain seen by tx pool", "block", newHead.Height, "hash", newHead.Hash())
					return
				}
			}
			reinject, accepted = types.TxDifference(discarded, included)
		}
	}
	// Inject any transactions discarded due to reorgs
	pool.Logger.Debug("Reinjecting stale transactions", "count", len(reinject))
	pool.AddTxs(reinject)
	// Remove any transactions accepted
	pool.Logger.Debug("Discard accepted transactions", "count", len(accepted))
	for _, t := range accepted {
		pool.remove(t.Key())
	}
}

// txLookup is used internally by TxPool to track transactions while allowing
// lookup without mutex contention.
//
// Note, although this type is properly protected against concurrent access, it
// is **not** a type that should ever be mutated or even exposed outside of the
// transaction pool, since its internal state is tightly coupled with the pools
// internal mechanisms. The sole purpose of the type is to permit out-of-bound
// peeking into the pool in TxPool.Get without having to acquire the widely scoped
// TxPool.mu mutex.
//
// This lookup set combines the notion of "local transactions", which is useful
// to build upper-level structure.
type txLookup struct {
	slots   int
	lock    sync.RWMutex
	locals  map[types.TxHash]types.Tx
	remotes map[types.TxHash]types.Tx
}

// newTxLookup returns a new txLookup structure.
func newTxLookup() *txLookup {
	return &txLookup{
		locals:  make(map[types.TxHash]types.Tx),
		remotes: make(map[types.TxHash]types.Tx),
	}
}

// Range calls f on each key and value present in the map. The callback passed
// should return the indicator whether the iteration needs to be continued.
// Callers need to specify which set (or both) to be iterated.
func (t *txLookup) Range(f func(hash types.TxHash, tx types.Tx, local bool) bool, local bool, remote bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	if local {
		for key, value := range t.locals {
			if !f(key, value, true) {
				return
			}
		}
	}
	if remote {
		for key, value := range t.remotes {
			if !f(key, value, false) {
				return
			}
		}
	}
}

// Get returns a transaction if it exists in the lookup, or nil if not found.
func (t *txLookup) Get(hash types.TxHash) types.Tx {
	t.lock.RLock()
	defer t.lock.RUnlock()

	if tx := t.locals[hash]; tx != nil {
		return tx
	}
	return t.remotes[hash]
}

// GetLocal returns a transaction if it exists in the lookup, or nil if not found.
func (t *txLookup) GetLocal(hash types.TxHash) types.Tx {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.locals[hash]
}

// GetRemote returns a transaction if it exists in the lookup, or nil if not found.
func (t *txLookup) GetRemote(hash types.TxHash) types.Tx {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.remotes[hash]
}

// Count returns the current number of transactions in the lookup.
func (t *txLookup) Count() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return len(t.locals) + len(t.remotes)
}

// LocalCount returns the current number of local transactions in the lookup.
func (t *txLookup) LocalCount() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return len(t.locals)
}

// RemoteCount returns the current number of remote transactions in the lookup.
func (t *txLookup) RemoteCount() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return len(t.remotes)
}

// Slots returns the current number of slots used in the lookup.
func (t *txLookup) Slots() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.slots
}

// Add adds a transaction to the lookup.
func (t *txLookup) Add(tx types.Tx, local bool) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.slots += numSlots(tx)
	// slotsGauge.Update(int64(t.slots))

	if local {
		t.locals[tx.Key()] = tx
	} else {
		t.remotes[tx.Key()] = tx
	}
}

// Remove removes a transaction from the lookup.
func (t *txLookup) Remove(hash types.TxHash) {
	t.lock.Lock()
	defer t.lock.Unlock()

	tx, ok := t.locals[hash]
	if !ok {
		tx, ok = t.remotes[hash]
	}
	if !ok {
		// log.Error("No transaction found to be deleted", "hash", hash)
		return
	}
	t.slots -= numSlots(tx)
	// slotsGauge.Update(int64(t.slots))

	delete(t.locals, hash)
	delete(t.remotes, hash)
}

func (t *txLookup) Flatten() types.Txs {
	txs := make(types.Txs, 0, len(t.locals))
	for _, tx := range t.locals {
		txs = append(txs, tx)
	}
	return txs
}

// // RemoteToLocals migrates the transactions belongs to the given locals to locals
// // set. The assumption is held the locals set is thread-safe to be used.
// func (t *txLookup) RemoteToLocals(locals *accountSet) int {
// 	t.lock.Lock()
// 	defer t.lock.Unlock()

// 	var migrated int
// 	for hash, tx := range t.remotes {
// 		if locals.containsTx(tx) {
// 			t.locals[hash] = tx
// 			delete(t.remotes, hash)
// 			migrated += 1
// 		}
// 	}
// 	return migrated
// }

// // RemotesBelowTip finds all remote transactions below the given tip threshold.
// func (t *txLookup) RemotesBelowTip(threshold *big.Int) types.Transactions {
// 	found := make(types.Transactions, 0, 128)
// 	t.Range(func(hash common.Hash, tx *types.Transaction, local bool) bool {
// 		if tx.GasTipCapIntCmp(threshold) < 0 {
// 			found = append(found, tx)
// 		}
// 		return true
// 	}, false, true) // Only iterate remotes
// 	return found
// }

// numSlots calculates the number of slots needed for a single transaction.
func numSlots(tx types.Tx) int {
	// return int((tx.Size() + txSlotSize - 1) / txSlotSize)
	return 1
}
