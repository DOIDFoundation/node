package config

// forks, zero means not forked yet, 1 means forked from genesis
type ForkNumber struct {
	value int64
}

func initDevnetForks() {
	OwnerFork.set(1)
	UncleFork.set(1)
}

func initTestnetForks() {
	OwnerFork.set(218000)
}

func (n *ForkNumber) Forked(height int64) bool {
	return n.value > 0 && height >= n.value
}

func (n *ForkNumber) set(height int64) {
	n.value = height
}
