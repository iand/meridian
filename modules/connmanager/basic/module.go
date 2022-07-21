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
	Low         int           `json:"low"`
	High        int           `json:"high"`
	GracePeriod time.Duration `json:"grace_period"`
}

func (b *Basic) ID() string { return "basic" }

func (b *Basic) ValidateModule() error {
	return nil
}

func (b *Basic) ProvideConnManager(ctx context.Context) (connmgr.ConnManager, error) {
	return bconnmgr.NewConnManager(b.Low, b.High, bconnmgr.WithGracePeriod(b.GracePeriod))
}
