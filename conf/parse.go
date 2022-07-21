package conf

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
)

func ParseAddrList(addrs []string) []multiaddr.Multiaddr {
	var mas []multiaddr.Multiaddr
	for _, addr := range addrs {
		ma, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			continue
		}
		mas = append(mas, ma)
	}
	return mas
}

func ParsePeerList(addrs []string) []peer.AddrInfo {
	var ais []peer.AddrInfo
	mas := ParseAddrList(addrs)
	for _, ma := range mas {
		ai, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			continue
		}
		ais = append(ais, *ai)
	}
	return ais
}
