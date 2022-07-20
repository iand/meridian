package basic

import (
	"context"

	"github.com/iand/meridian/conf"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-ipfs-blockstore"
)

func init() {
	conf.RegisterModule(conf.ModuleCategoryBlockstore, &Basic{
		HashOnRead: true,
	})
}

type Basic struct {
	HashOnRead bool `json:"hash_on_read"`
}

func (b *Basic) ID() string { return "basic" }

func (b *Basic) ValidateModule() error {
	return nil
}

func (b *Basic) ProvideBlockstore(ctx context.Context, ds datastore.Batching) (blockstore.Blockstore, error) {
	bs := blockstore.NewBlockstore(ds)
	bs = blockstore.NewIdStore(bs)
	bs.HashOnRead(b.HashOnRead)

	return bs, nil
}
