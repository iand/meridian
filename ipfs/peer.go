package ipfs

import (
	"context"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/ipfs/go-bitswap"
	"github.com/ipfs/go-bitswap/network"
	blockservice "github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger2"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	offline "github.com/ipfs/go-ipfs-exchange-offline"
	pin "github.com/ipfs/go-ipfs-pinner"
	"github.com/ipfs/go-ipfs-pinner/dspinner"
	provider "github.com/ipfs/go-ipfs-provider"
	"github.com/ipfs/go-ipfs-provider/queue"
	"github.com/ipfs/go-ipfs-provider/simple"
	ipld "github.com/ipfs/go-ipld-format"
	logging "github.com/ipfs/go-log/v2"
	"github.com/ipfs/go-merkledag"
	metrics "github.com/ipfs/go-metrics-interface"
	"github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	routing "github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/fullrt"
	libp2ptls "github.com/libp2p/go-libp2p-tls"
	connmgr "github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/multiformats/go-multiaddr"
	multihash "github.com/multiformats/go-multihash"
)

var logger = logging.Logger("ipfs")

var BootstrapPeers = []peer.AddrInfo{
	mustParseAddr("/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN"),
	mustParseAddr("/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa"),
	mustParseAddr("/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb"),
	mustParseAddr("/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt"),
}

func mustParseAddr(addr string) peer.AddrInfo {
	ma, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		panic(fmt.Sprintf("failed to parse bootstrap address: %v", err))
	}

	ai, err := peer.AddrInfoFromP2pAddr(ma)
	if err != nil {
		panic(fmt.Sprintf("failed to create address info: %v", err))
	}

	return *ai
}

var defaultReprovideInterval = 12 * time.Hour

// Config wraps configuration options for the Peer.
type PeerConfig struct {
	Name              string
	Offline           bool
	ReprovideInterval time.Duration
	DatastorePath     string
	FileSystemPath    string
	ListenAddr        string
	Libp2pKeyFile     string
	ManifestPath      string
}

type Peer struct {
	offline           bool
	reprovideInterval time.Duration
	listenAddr        multiaddr.Multiaddr
	peerKey           crypto.PrivKey
	datastorePath     string
	fileSystemPath    string
	manifestPath      string
	builder           cid.Builder

	host host.Host
	dht  routing.Routing
	ds   datastore.Batching

	dag    ipld.DAGService // (consider ipld.BufferedDAG)
	bs     blockstore.GCBlockstore
	pinner pin.Pinner

	mu       sync.Mutex // guards writes to bserv, reprovider fields
	bserv    blockservice.BlockService
	provider provider.System
}

func NewPeer(cfg *PeerConfig) (*Peer, error) {
	// Create a temporary context to hold metrics metadata
	ctx := metrics.CtxScope(context.Background(), cfg.Name)

	p := new(Peer)

	if err := p.applyConfig(cfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	if err := p.setupDatastore(); err != nil {
		return nil, fmt.Errorf("setup datastore: %w", err)
	}

	if err := p.setupBlockstore(ctx); err != nil {
		return nil, fmt.Errorf("setup blockstore: %w", err)
	}

	if !p.offline {
		if err := p.setupLibp2p(ctx); err != nil {
			return nil, fmt.Errorf("setup libp2p: %w", err)
		}

		if err := p.setupDHT(ctx); err != nil {
			return nil, fmt.Errorf("setup dht: %w", err)
		}

		if err := p.bootstrap(ctx, BootstrapPeers); err != nil {
			return nil, fmt.Errorf("bootstrap: %w", err)
		}
	}

	if err := p.setupBlockService(ctx); err != nil {
		return nil, fmt.Errorf("setup blockservice: %w", err)
	}

	if err := p.setupReprovider(ctx); err != nil {
		return nil, fmt.Errorf("setup reprovider: %w", err)
	}

	if err := p.setupDAGService(); err != nil {
		p.Close()
		return nil, fmt.Errorf("setup dagservice: %w", err)
	}

	return p, nil
}

func (p *Peer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.provider != nil {
		if err := p.provider.Close(); err != nil {
			return fmt.Errorf("reprovider: %w", err)
		}
		p.provider = nil
	}
	if p.bserv != nil {
		if err := p.bserv.Close(); err != nil {
			return fmt.Errorf("block service: %w", err)
		}
		p.bserv = nil
	}

	return nil
}

func (p *Peer) applyConfig(cfg *PeerConfig) error {
	if cfg == nil {
		cfg = &PeerConfig{}
	}

	p.offline = cfg.Offline
	p.manifestPath = cfg.ManifestPath

	if cfg.ReprovideInterval == 0 {
		p.reprovideInterval = defaultReprovideInterval
	} else {
		p.reprovideInterval = cfg.ReprovideInterval
	}

	if cfg.DatastorePath == "" {
		return fmt.Errorf("missing datastore path")
	}
	p.datastorePath = cfg.DatastorePath

	if cfg.FileSystemPath == "" {
		return fmt.Errorf("missing file system path")
	}
	p.fileSystemPath = cfg.FileSystemPath

	var err error
	p.listenAddr, err = multiaddr.NewMultiaddr(cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen addr: %w", err)
	}

	if cfg.Libp2pKeyFile == "" {
		return fmt.Errorf("missing libp2p keyfile")
	}
	p.peerKey, err = loadOrInitPeerKey(cfg.Libp2pKeyFile)
	if err != nil {
		return fmt.Errorf("key file: %w", err)
	}

	// Set up a consistent cid builder
	const hashfunc = "sha2-256"
	prefix, err := merkledag.PrefixForCidVersion(1)
	if err != nil {
		return fmt.Errorf("bad CID Version: %s", err)
	}

	hashFunCode, ok := multihash.Names[hashfunc]
	if !ok {
		return fmt.Errorf("unrecognized hash function: %s", hashfunc)
	}
	prefix.MhType = hashFunCode
	prefix.MhLength = -1

	p.builder = &prefix

	return nil
}

func (p *Peer) setupDatastore() error {
	logger.Debugf("setting up ipfs datastore at %s", p.datastorePath)
	opts := badger.DefaultOptions

	ds, err := badger.NewDatastore(p.datastorePath, &opts)
	if err != nil {
		return fmt.Errorf("new datastore: %w", err)
	}
	p.ds = ds
	return nil
}

func (p *Peer) setupBlockstore(ctx context.Context) error {
	logger.Debug("setting up ipfs blockstore")

	bs := blockstore.NewBlockstore(p.ds)
	bs = blockstore.NewIdStore(bs)
	bs.HashOnRead(true)

	cbs, err := blockstore.CachedBlockstore(ctx, bs, blockstore.DefaultCacheOpts())
	if err != nil {
		return fmt.Errorf("new cached blockstore: %w", err)
	}

	gclocker := blockstore.NewGCLocker()
	p.bs = blockstore.NewGCBlockstore(cbs, gclocker)

	return nil
}

func (p *Peer) setupBlockService(ctx context.Context) error {
	logger.Debug("setting up ipfs block service")
	if p.offline {
		p.bserv = blockservice.New(p.bs, offline.Exchange(p.bs))
		return nil
	}

	bswapnet := network.NewFromIpfsHost(p.host, p.dht)

	bswap := bitswap.New(ctx, bswapnet, p.bs,
		bitswap.ProvideEnabled(false),
		bitswap.EngineBlockstoreWorkerCount(600),
		bitswap.TaskWorkerCount(600),
		bitswap.MaxOutstandingBytesPerPeer(5<<20),
	)

	bserv := blockservice.New(p.bs, bswap)
	p.mu.Lock()
	p.bserv = bserv
	p.mu.Unlock()

	return nil
}

func (p *Peer) setupPinner(ctx context.Context) error {
	pinner, err := dspinner.New(ctx, p.ds, p.dag)
	if err != nil {
		return fmt.Errorf("new pinner: %w", err)
	}

	p.pinner = pinner
	return nil
}

func (p *Peer) setupDAGService() error {
	p.dag = merkledag.NewDAGService(p.bserv)
	return nil
}

func (p *Peer) setupReprovider(ctx context.Context) error {
	logger.Debug("setting up reprovider")
	if p.offline || p.reprovideInterval < 0 {
		p.provider = provider.NewOfflineProvider()
		return nil
	}

	queue, err := queue.NewQueue(ctx, "repro", p.ds)
	if err != nil {
		return err
	}

	prov := simple.NewProvider(
		ctx,
		queue,
		p.dht,
	)

	reprov := simple.NewReprovider(
		ctx,
		p.reprovideInterval,
		p.dht,
		simple.NewBlockstoreProvider(p.bs),
	)

	reprovider := provider.NewSystem(prov, reprov)
	reprovider.Run()

	p.mu.Lock()
	p.provider = reprovider
	p.mu.Unlock()

	return nil
}

func (p *Peer) bootstrap(ctx context.Context, peers []peer.AddrInfo) error {
	logger.Info("bootstrapping ipfs node")
	connected := make(chan struct{}, len(peers))

	var wg sync.WaitGroup
	for _, pinfo := range peers {
		// h.Peerstore().AddAddrs(pinfo.ID, pinfo.Addrs, peerstore.PermanentAddrTTL)
		wg.Add(1)
		go func(pinfo peer.AddrInfo) {
			defer wg.Done()
			err := p.host.Connect(ctx, pinfo)
			if err != nil {
				logger.Warn(err)
				return
			}
			logger.Debugf("Connected to %s", pinfo.ID)
			connected <- struct{}{}
		}(pinfo)
	}

	wg.Wait()
	close(connected)

	i := 0
	for range connected {
		i++
	}

	logger.Debugf("connected to %d peers", len(peers))

	err := p.dht.Bootstrap(context.TODO())
	if err != nil {
		return fmt.Errorf("dht bootstrap: %w", err)
	}

	p.logHostAddresses()

	return nil
}

func (p *Peer) setupLibp2p(ctx context.Context) error {
	var err error

	connmgr, err := connmgr.NewConnManager(100, 600, connmgr.WithGracePeriod(time.Minute))
	if err != nil {
		return fmt.Errorf("libp2p: %w", err)
	}

	finalOpts := []libp2p.Option{
		libp2p.Identity(p.peerKey),
		libp2p.ListenAddrs(p.listenAddr),
		libp2p.NATPortMap(),
		libp2p.ConnectionManager(connmgr),
		libp2p.EnableAutoRelay(),
		libp2p.EnableNATService(),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.DefaultTransports,
	}

	h, err := libp2p.New(
		finalOpts...,
	)
	if err != nil {
		return fmt.Errorf("new libp2p: %w", err)
	}

	p.host = h

	return nil
}

func (p *Peer) setupDHT(ctx context.Context) error {
	dhtopts := fullrt.DHTOption(
		dht.Datastore(p.ds),
		dht.BootstrapPeers(BootstrapPeers...),
		dht.BucketSize(20),
	)

	frt, err := fullrt.NewFullRT(p.host, dht.DefaultPrefix, dhtopts)
	if err != nil {
		return fmt.Errorf("new fullrt: %w", err)
	}

	p.dht = frt
	return nil
}

func (p *Peer) logHostAddresses() {
	if p.offline {
		logger.Debugf("not listening, running in offline mode")
		return
	}

	var lisAddrs []string
	ifaceAddrs, err := p.host.Network().InterfaceListenAddresses()
	if err != nil {
		logger.Errorf("failed to read listening addresses: %s", err)
	}
	for _, addr := range ifaceAddrs {
		lisAddrs = append(lisAddrs, addr.String())
	}
	sort.Strings(lisAddrs)
	for _, addr := range lisAddrs {
		logger.Debugf("listening on %s", addr)
	}

	var addrs []string
	for _, addr := range p.host.Addrs() {
		addrs = append(addrs, addr.String())
	}
	sort.Strings(addrs)
	for _, addr := range addrs {
		logger.Debugf("announcing %s", addr)
	}
}

func loadOrInitPeerKey(kf string) (crypto.PrivKey, error) {
	data, err := ioutil.ReadFile(kf)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		keyDir := filepath.Dir(kf)
		if err := os.MkdirAll(keyDir, os.ModePerm); err != nil {
			return nil, fmt.Errorf("mkdir %q: %w", keyDir, err)
		}

		k, _, err := crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			return nil, err
		}

		data, err := crypto.MarshalPrivateKey(k)
		if err != nil {
			return nil, err
		}

		if err := ioutil.WriteFile(kf, data, 0o600); err != nil {
			return nil, err
		}

		return k, nil
	}
	return crypto.UnmarshalPrivateKey(data)
}

func (p *Peer) Datastore() datastore.Batching {
	return p.ds
}

func (p *Peer) DAGService() ipld.DAGService {
	return p.dag
}

func (p *Peer) GCBlockstore() blockstore.GCBlockstore {
	return p.bs
}

func (p *Peer) Pinner() pin.Pinner {
	return p.pinner
}

func (p *Peer) BlockService() blockservice.BlockService {
	return p.bserv
}

func (p *Peer) Routing() routing.Routing {
	return p.dht
}

func (p *Peer) ProviderSystem() provider.System {
	return p.provider
}
