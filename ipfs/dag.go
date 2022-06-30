package ipfs

import (
	"context"

	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	pin "github.com/ipfs/go-ipfs-pinner"
	ipld "github.com/ipfs/go-ipld-format"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
)

var _ coreiface.APIDagService = (*PinningDagService)(nil)

type PinningDagService struct {
	ipld.DAGService
	ds datastore.Datastore
	pa *PinningAdder
}

func NewPinningDagService(bs blockstore.GCBlockstore, dag ipld.DAGService, pinner pin.Pinner) *PinningDagService {
	return &PinningDagService{
		DAGService: dag,
		pa: &PinningAdder{
			dag:    dag,
			bs:     bs,
			pinner: pinner,
		},
	}
}

func (s *PinningDagService) Pinning() ipld.NodeAdder {
	return s.pa
}

var _ ipld.NodeAdder = (*PinningAdder)(nil)

type PinningAdder struct {
	bs     blockstore.GCBlockstore
	dag    ipld.DAGService
	pinner pin.Pinner
}

func (a *PinningAdder) Add(ctx context.Context, nd ipld.Node) error {
	defer a.bs.PinLock(ctx).Unlock(ctx)

	if err := a.dag.Add(ctx, nd); err != nil {
		return err
	}

	a.pinner.PinWithMode(nd.Cid(), pin.Recursive)

	return a.pinner.Flush(ctx)
}

func (a *PinningAdder) AddMany(ctx context.Context, nds []ipld.Node) error {
	defer a.bs.PinLock(ctx).Unlock(ctx)

	if err := a.dag.AddMany(ctx, nds); err != nil {
		return err
	}

	cids := cid.NewSet()

	for _, nd := range nds {
		c := nd.Cid()
		if cids.Visit(c) {
			a.pinner.PinWithMode(c, pin.Recursive)
		}
	}

	return a.pinner.Flush(ctx)
}
