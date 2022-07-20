package badger2

import (
	"context"
	"fmt"

	"github.com/iand/meridian/conf"
	"github.com/ipfs/go-datastore"
)

func init() {
	conf.RegisterModule(conf.ModuleCategoryDatastoreWrapper, &Log{})
}

type Log struct {
	Name string `json:"name"`
}

func (l *Log) ID() string { return "log" }

func (l *Log) ValidateModule() error {
	if l.Name == "" {
		return fmt.Errorf("name must be specified")
	}
	return nil
}

func (l *Log) WrapDatastore(ctx context.Context, ds datastore.Batching) (datastore.Batching, error) {
	return datastore.NewLogDatastore(ds, l.Name), nil
}
