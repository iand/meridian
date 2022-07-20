package badger2

import (
	"context"
	"fmt"

	"github.com/iand/meridian/conf"
	"github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger2"
)

func init() {
	conf.RegisterModule(conf.ModuleCategoryDatastore, &Badger{})
}

type Badger struct {
	Path string `json:"path"`
}

func (b *Badger) ID() string { return "badger" }

func (b *Badger) ValidateModule() error {
	if b.Path == "" {
		return fmt.Errorf("path must be specified")
	}
	return nil
}

func (b *Badger) ProvideDatastore(ctx context.Context) (datastore.Batching, error) {
	opts := badger.DefaultOptions

	ds, err := badger.NewDatastore(b.Path, &opts)
	if err != nil {
		return nil, fmt.Errorf("new datastore: %w", err)
	}

	return ds, nil
}
