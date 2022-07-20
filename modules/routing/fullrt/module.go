package fullrt

import (
	"context"
	"fmt"

	"github.com/iand/meridian/conf"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/fullrt"
)

func init() {
	conf.RegisterModule(conf.ModuleCategoryDatastoreWrapper, &Full{})
}

type Full struct{}

func (f *Full) ID() string { return "fullrt" }

func (f *Full) ValidateModule() error {
	return nil
}

func (f *Full) ProvideRouting(ctx context.Context, ds datastore.Batching, h host.Host) (routing.Routing, error) {
	dhtopts := fullrt.DHTOption(
		dht.Datastore(ds),
		// dht.BootstrapPeers(BootstrapPeers...),
		dht.BucketSize(20),
	)

	frt, err := fullrt.NewFullRT(h, dht.DefaultPrefix, dhtopts)
	if err != nil {
		return nil, fmt.Errorf("new fullrt: %w", err)
	}

	return frt, nil
}
