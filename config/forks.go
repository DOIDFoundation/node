package config

func initDevnetForks() {
	OwnerFork = 0
}

func initTestnetForks() {
	OwnerFork = 104870
}

func isFork(height int64, fork int64) bool {
	return fork >= 0 && height >= fork
}

func IsOwnerFork(height int64) bool {
	return isFork(height, OwnerFork)
}
