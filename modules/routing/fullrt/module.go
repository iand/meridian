package fullrt

import (
	"context"
	"fmt"

	"github.com/iand/meridian/conf"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/fullrt"
	"github.com/libp2p/go-libp2p-record"
)

func init() {
	conf.RegisterModule(conf.ModuleCategoryRouting, &Full{
		BootstrapPeers: []string{
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
		},
	})
}

type Full struct {
	// TODO: more configuration options
	BootstrapPeers []string `json:"bootstrap_peers"`
}

func (*Full) ID() string { return "fullrt" }

func (*Full) ValidateModule() error {
	return nil
}

func (f *Full) ProvideRouting(ctx context.Context, ds datastore.Batching, h host.Host, v record.Validator) (routing.Routing, error) {
	dhtopts := fullrt.DHTOption(
		dht.Datastore(ds),
		dht.BootstrapPeers(conf.ParsePeerList(f.BootstrapPeers)...),
		dht.BucketSize(20),
	)

	frt, err := fullrt.NewFullRT(h, dht.DefaultPrefix, dhtopts)
	if err != nil {
		return nil, fmt.Errorf("new fullrt: %w", err)
	}

	return frt, nil
}
