package fullrt

import (
	"context"
	"fmt"

	"github.com/iand/meridian/conf"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-record"
)

func init() {
	conf.RegisterModule(conf.ModuleCategoryRouting, &Dht{})
}

type Dht struct {
	// TODO: configuration options
}

func (*Dht) ID() string { return "dht" }

func (*Dht) ValidateModule() error {
	return nil
}

func (*Dht) ProvideRouting(ctx context.Context, ds datastore.Batching, h host.Host, v record.Validator) (routing.Routing, error) {
	rt, err := dht.New(ctx, h, dht.Datastore(ds))
	if err != nil {
		return nil, fmt.Errorf("new dht: %w", err)
	}

	return rt, nil
}
