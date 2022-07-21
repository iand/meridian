package tiered

import (
	"context"

	"github.com/iand/meridian/conf"
	"github.com/libp2p/go-libp2p-core/routing"
	"github.com/libp2p/go-libp2p-record"
	"github.com/libp2p/go-libp2p-routing-helpers"
)

func init() {
	conf.RegisterModule(conf.ModuleCategoryRoutingComposer, &Tiered{})
}

type Tiered struct{}

func (*Tiered) ID() string { return "tiered" }

func (*Tiered) ValidateModule() error {
	return nil
}

func (*Tiered) ComposeRouting(ctx context.Context, rs []routing.Routing, v record.Validator) (routing.Routing, error) {
	return routinghelpers.Tiered{Routers: rs, Validator: v}, nil
}
