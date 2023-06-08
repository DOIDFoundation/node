package network

import (
	maddr "github.com/multiformats/go-multiaddr"
	"strings"
)

// A new type we need for writing a custom flag parser
type addrList []maddr.Multiaddr

func (al *addrList) String() string {
	strs := make([]string, len(*al))
	for i, addr := range *al {
		strs[i] = addr.String()
	}
	return strings.Join(strs, ",")
}

func (al *addrList) Set(value string) error {
	addr, err := maddr.NewMultiaddr(value)
	if err != nil {
		return err
	}
	*al = append(*al, addr)
	return nil
}

// Config defines the top level configuration for a network
type Config struct {
	ListenAddresses  string
	RendezvousString string
	BootstrapPeers   addrList
	ProtocolID       string
}

// DefaultConfig returns a default configuration for a  network
var DefaultConfig = Config{}