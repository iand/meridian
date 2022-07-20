package basic

import (
	"context"
	"sync"

	"github.com/iand/meridian/conf"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
)

func init() {
	conf.RegisterModule(conf.ModuleCategoryBootstrapper, &Basic{})
}

type Basic struct{}

func (b *Basic) ID() string { return "basic" }

func (b *Basic) ValidateModule() error {
	return nil
}

func (b *Basic) BootstrapHost(ctx context.Context, h host.Host, peers []peer.AddrInfo) error {
	connected := make(chan struct{}, len(peers))

	var wg sync.WaitGroup
	for _, pinfo := range peers {
		wg.Add(1)
		go func(pinfo peer.AddrInfo) {
			defer wg.Done()
			err := h.Connect(ctx, pinfo)
			if err != nil {
				return
			}
			connected <- struct{}{}
		}(pinfo)
	}

	wg.Wait()
	close(connected)

	i := 0
	for range connected {
		i++
	}
	return nil
}
