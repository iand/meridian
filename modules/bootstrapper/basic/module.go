package basic

import (
	"context"
	"sync"

	"github.com/iand/meridian/conf"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
)

func init() {
	conf.RegisterModule(conf.ModuleCategoryBootstrapper, &Basic{
		BootstrapPeers: []string{
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
		},
	})
}

type Basic struct {
	BootstrapPeers []string `json:"bootstrap_peers"`
}

func (b *Basic) ID() string { return "basic" }

func (b *Basic) ValidateModule() error {
	return nil
}

func (b *Basic) BootstrapHost(ctx context.Context, h host.Host) error {
	peers := conf.ParsePeerList(b.BootstrapPeers)

	var wg sync.WaitGroup
	for _, pinfo := range peers {
		wg.Add(1)
		go func(pinfo peer.AddrInfo) {
			defer wg.Done()
			err := h.Connect(ctx, pinfo)
			if err != nil {
				return
			}
		}(pinfo)
	}

	wg.Wait()
	return nil
}
