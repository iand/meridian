package basic

import (
	"context"
	"time"

	"github.com/iand/meridian/conf"
	"github.com/libp2p/go-libp2p-core/connmgr"
	bconnmgr "github.com/libp2p/go-libp2p/p2p/net/connmgr"
)

func init() {
	conf.RegisterModule(conf.ModuleCategoryConnManager, &Basic{
		Low:         100,
		High:        600,
		GracePeriod: time.Minute,
	})
}

type Basic struct {
	Low         int
	High        int
	GracePeriod time.Duration
}

func (b *Basic) ID() string { return "basic" }

func (b *Basic) ValidateModule() error {
	return nil
}

func (b *Basic) ProvideConnManager(ctx context.Context) (connmgr.ConnManager, error) {
	return bconnmgr.NewConnManager(b.Low, 600, bconnmgr.WithGracePeriod(b.GracePeriod))
}
