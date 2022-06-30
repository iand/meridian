package ipfs

import (
	"context"
	"fmt"
	gopath "path"

	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-fetcher"
	bsfetcher "github.com/ipfs/go-fetcher/impl/blockservice"
	"github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-namesys"
	"github.com/ipfs/go-namesys/resolve"
	ipfspath "github.com/ipfs/go-path"
	ipfspathresolver "github.com/ipfs/go-path/resolver"
	"github.com/ipfs/go-unixfsnode"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	path "github.com/ipfs/interface-go-ipfs-core/path"
	dagpb "github.com/ipld/go-codec-dagpb"
	"github.com/ipld/go-ipld-prime"
	basicnode "github.com/ipld/go-ipld-prime/node/basic"
	"github.com/ipld/go-ipld-prime/schema"
	madns "github.com/multiformats/go-multiaddr-dns"
)

type Gateway struct {
	p                    *Peer
	pds                  *PinningDagService
	ufsapi               *UnixfsAPI
	blockapi             *BlockAPI
	ipldFetcherFactory   fetcher.Factory
	unixFSFetcherFactory fetcher.Factory
	namesys              namesys.NameSystem
}

func NewGateway(p *Peer) (*Gateway, error) {
	g := &Gateway{
		p: p,
	}
	if err := g.setup(); err != nil {
		return nil, fmt.Errorf("setup: %w", err)
	}

	g.pds = NewPinningDagService(p.GCBlockstore(), p.DAGService(), p.Pinner())
	g.blockapi = NewBlockAPI(p.GCBlockstore(), p.BlockService(), p.Pinner(), g)
	g.ufsapi = NewUnixfsAPI(p.Datastore(), p.DAGService(), p.GCBlockstore(), p.Pinner(), p.ProviderSystem(), g, p.BlockService())

	ipldFetcher := bsfetcher.NewFetcherConfig(p.BlockService())
	ipldFetcher.PrototypeChooser = dagpb.AddSupportToChooser(func(lnk ipld.Link, lnkCtx ipld.LinkContext) (ipld.NodePrototype, error) {
		if tlnkNd, ok := lnkCtx.LinkNode.(schema.TypedLinkNode); ok {
			return tlnkNd.LinkTargetNodePrototype(), nil
		}
		return basicnode.Prototype.Any, nil
	})

	g.ipldFetcherFactory = ipldFetcher
	g.unixFSFetcherFactory = ipldFetcher.WithReifier(unixfsnode.Reify)

	return g, nil
}

func (g *Gateway) setup() error {
	// TODO better resolver
	rslvr, err := madns.NewResolver()
	if err != nil {
		return fmt.Errorf("new resolver: %w", err)
	}

	opts := []namesys.Option{
		namesys.WithDatastore(g.p.Datastore()),
		namesys.WithDNSResolver(rslvr),
	}

	g.namesys, err = namesys.NewNameSystem(g.p.Routing(), opts...)
	if err != nil {
		return fmt.Errorf("new name system: %w", err)
	}

	return nil
}

// Unixfs returns an implementation of Unixfs API
func (g *Gateway) Unixfs() coreiface.UnixfsAPI {
	return g.ufsapi
}

// Block returns an implementation of Block API
func (g *Gateway) Block() coreiface.BlockAPI {
	return g.blockapi
}

// Dag returns an implementation of Dag API
func (g *Gateway) Dag() coreiface.APIDagService {
	return g.pds
}

type BlockStat struct {
	path path.Resolved
	size int
}

func (g *Gateway) ResolvePath(ctx context.Context, pth path.Path) (path.Resolved, error) {
	if _, ok := pth.(path.Resolved); ok {
		return pth.(path.Resolved), nil
	}
	if err := pth.IsValid(); err != nil {
		return nil, fmt.Errorf("path: %w", err)
	}

	ipath := ipfspath.Path(pth.String())
	ipath, err := resolve.ResolveIPNS(ctx, g.namesys, ipath)
	if err == resolve.ErrNoNamesys {
		return nil, coreiface.ErrOffline
	} else if err != nil {
		return nil, fmt.Errorf("resolve ipns: %w", err)
	}

	if ipath.Segments()[0] != "ipfs" && ipath.Segments()[0] != "ipld" {
		return nil, fmt.Errorf("unsupported path namespace: %s", pth.Namespace())
	}

	var dataFetcher fetcher.Factory
	if ipath.Segments()[0] == "ipld" {
		dataFetcher = g.ipldFetcherFactory
	} else {
		dataFetcher = g.unixFSFetcherFactory
	}
	resolver := ipfspathresolver.NewBasicResolver(dataFetcher)

	node, rest, err := resolver.ResolveToLastNode(ctx, ipath)
	if err != nil {
		return nil, fmt.Errorf("resolve to last node: %w", err)
	}

	root, err := cid.Parse(ipath.Segments()[1])
	if err != nil {
		return nil, fmt.Errorf("parse root cid: %w", err)
	}

	return path.NewResolvedPath(ipath, node, root, gopath.Join(rest...)), nil
}

func (g *Gateway) ResolveNode(ctx context.Context, pth path.Path) (format.Node, error) {
	rp, err := g.ResolvePath(ctx, pth)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	node, err := g.p.DAGService().Get(ctx, rp.Cid())
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}
	return node, nil
}
