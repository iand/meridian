package parallel

import (
	"context"

	"github.com/iand/meridian/conf"
	"github.com/libp2p/go-libp2p-core/routing"
	"github.com/libp2p/go-libp2p-record"
	"github.com/libp2p/go-libp2p-routing-helpers"
)

func init() {
	conf.RegisterModule(conf.ModuleCategoryRoutingComposer, &Parallel{})
}

type Parallel struct{}

func (*Parallel) ID() string { return "parallel" }

func (*Parallel) ValidateModule() error {
	return nil
}

func (*Parallel) ComposeRouting(ctx context.Context, rs []routing.Routing, v record.Validator) (routing.Routing, error) {
	return routinghelpers.Parallel{Routers: rs, Validator: v}, nil
}
