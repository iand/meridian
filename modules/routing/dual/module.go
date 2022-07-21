package dual

import (
	"context"
	"fmt"

	"github.com/iand/meridian/conf"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p-record"
)

func init() {
	conf.RegisterModule(conf.ModuleCategoryRouting, &Dual{})
}

type Dual struct {
	// TODO: configuration options
}

func (*Dual) ID() string { return "dual" }

func (*Dual) ValidateModule() error {
	return nil
}

func (*Dual) ProvideRouting(ctx context.Context, ds datastore.Batching, h host.Host, v record.Validator) (routing.Routing, error) {
	rt, err := dual.New(ctx, h, dual.DHTOption(dht.Datastore(ds)))
	if err != nil {
		return nil, fmt.Errorf("new dual dht: %w", err)
	}

	return rt, nil
}
