package null

import (
	"context"

	"github.com/iand/meridian/conf"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	"github.com/libp2p/go-libp2p-record"
	"github.com/libp2p/go-libp2p-routing-helpers"
)

func init() {
	conf.RegisterModule(conf.ModuleCategoryRouting, &Null{})
}

type Null struct{}

func (*Null) ID() string { return "null" }

func (*Null) ValidateModule() error {
	return nil
}

func (*Null) ProvideRouting(ctx context.Context, ds datastore.Batching, h host.Host, v record.Validator) (routing.Routing, error) {
	return routinghelpers.Null{}, nil
}
