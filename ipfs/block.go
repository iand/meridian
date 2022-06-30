package ipfs

import (
	"bytes"
	"context"
	"errors"
	"io"

	blocks "github.com/ipfs/go-block-format"
	blockservice "github.com/ipfs/go-blockservice"
	cid "github.com/ipfs/go-cid"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	pin "github.com/ipfs/go-ipfs-pinner"
	util "github.com/ipfs/go-ipfs/blocks/blockstoreutil"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	caopts "github.com/ipfs/interface-go-ipfs-core/options"
	path "github.com/ipfs/interface-go-ipfs-core/path"
)

var _ coreiface.BlockAPI = (*BlockAPI)(nil)

type BlockAPI struct {
	bs       blockstore.GCBlockstore
	bserv    blockservice.BlockService
	pinner   pin.Pinner
	resolver Resolver
}

func NewBlockAPI(bs blockstore.GCBlockstore, bserv blockservice.BlockService, pinner pin.Pinner, resolver Resolver) *BlockAPI {
	return &BlockAPI{
		bs:       bs,
		bserv:    bserv,
		pinner:   pinner,
		resolver: resolver,
	}
}

func (ba *BlockAPI) Put(ctx context.Context, src io.Reader, opts ...caopts.BlockPutOption) (coreiface.BlockStat, error) {
	settings, err := caopts.BlockPutOptions(opts...)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(src)
	if err != nil {
		return nil, err
	}

	bcid, err := settings.CidPrefix.Sum(data)
	if err != nil {
		return nil, err
	}

	b, err := blocks.NewBlockWithCid(data, bcid)
	if err != nil {
		return nil, err
	}

	if settings.Pin {
		defer ba.bs.PinLock(ctx).Unlock(ctx)
	}

	err = ba.bserv.AddBlock(ctx, b)
	if err != nil {
		return nil, err
	}

	if settings.Pin {
		ba.pinner.PinWithMode(b.Cid(), pin.Recursive)
		if err := ba.pinner.Flush(ctx); err != nil {
			return nil, err
		}
	}

	return &BlockStat{path: path.IpldPath(b.Cid()), size: len(data)}, nil
}

func (ba *BlockAPI) Get(ctx context.Context, pth path.Path) (io.Reader, error) {
	rp, err := ba.resolver.ResolvePath(ctx, pth)
	if err != nil {
		return nil, err
	}

	b, err := ba.bserv.GetBlock(ctx, rp.Cid())
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(b.RawData()), nil
}

func (ba *BlockAPI) Rm(ctx context.Context, pth path.Path, opts ...caopts.BlockRmOption) error {
	rp, err := ba.resolver.ResolvePath(ctx, pth)
	if err != nil {
		return err
	}

	settings, err := caopts.BlockRmOptions(opts...)
	if err != nil {
		return err
	}
	cids := []cid.Cid{rp.Cid()}
	o := util.RmBlocksOpts{Force: settings.Force}

	out, err := util.RmBlocks(ctx, ba.bs, ba.pinner, cids, o)
	if err != nil {
		return err
	}

	select {
	case res, ok := <-out:
		if !ok {
			return nil
		}

		remBlock, ok := res.(*util.RemovedBlock)
		if !ok {
			return errors.New("got unexpected output from util.RmBlocks")
		}

		if remBlock.Error != nil {
			return remBlock.Error
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (ba *BlockAPI) Stat(ctx context.Context, pth path.Path) (coreiface.BlockStat, error) {
	rp, err := ba.resolver.ResolvePath(ctx, pth)
	if err != nil {
		return nil, err
	}

	b, err := ba.bserv.GetBlock(ctx, rp.Cid())
	if err != nil {
		return nil, err
	}

	return &BlockStat{
		path: path.IpldPath(b.Cid()),
		size: len(b.RawData()),
	}, nil
}

func (bs *BlockStat) Size() int {
	return bs.size
}

func (bs *BlockStat) Path() path.Resolved {
	return bs.path
}
