package conf

import (
	"context"
	"fmt"
	"sync"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-ipfs-blockstore"
	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
)

const (
	ModuleCategoryDatastore         = "datastore"
	ModuleCategoryDatastoreWrapper  = "datastore_wrapper"
	ModuleCategoryBlockstore        = "blockstore"
	ModuleCategoryBlockstoreWrapper = "blockstore_wrapper"
	ModuleCategoryBootstrapper      = "bootstrapper"
	ModuleCategoryRouting           = "routing"
	ModuleCategoryConnManager       = "connection_manager"
)

type Module interface {
	//	ID returns a short identifier for the module, which should not contain whitespace
	ID() string

	// ValidateModule checks that the module's config is valid and returns an error if not
	ValidateModule() error
}

type DatastoreProvider interface {
	ProvideDatastore(context.Context) (datastore.Batching, error)
}

type DatastoreWrapper interface {
	WrapDatastore(context.Context, datastore.Batching) (datastore.Batching, error)
}

type BlockstoreProvider interface {
	ProvideBlockstore(context.Context, datastore.Batching) (blockstore.Blockstore, error)
}

type BlockstoreWrapper interface {
	WrapBlockstore(context.Context, blockstore.Blockstore) (blockstore.Blockstore, error)
}

type Bootstrapper interface {
	BootstrapHost(context.Context, host.Host, []peer.AddrInfo) error
}

type RoutingProvider interface {
	ProvideRouting(context.Context) (routing.Routing, error)
}

type ConnManagerProvider interface {
	ProvideConnManager(context.Context) (connmgr.ConnManager, error)
}

var (
	modules   = map[string]map[string]Module{}
	modulesMu sync.Mutex
)

// RegisterModule registers a module for a particular category.
// The passed module should be an empty instance that config
// can be unmarshaled into
func RegisterModule(category string, module Module) {
	if module == nil {
		panic(fmt.Sprintf("failed to register nil module for category %q", category))
	}

	switch category {
	case ModuleCategoryDatastore:
		// module must implement the correct interface
		if _, ok := module.(DatastoreProvider); !ok {
			panic(fmt.Sprintf("failed to register module %q for category %q since it does not implement the DatastoreProvider interface", module.ID(), category))
		}
	case ModuleCategoryDatastoreWrapper:
		// module must implement the correct interface
		if _, ok := module.(DatastoreWrapper); !ok {
			panic(fmt.Sprintf("failed to register module %q for category %q since it does not implement the DatastoreWrapper interface", module.ID(), category))
		}
	case ModuleCategoryBlockstore:
		// module must implement the correct interface
		if _, ok := module.(BlockstoreProvider); !ok {
			panic(fmt.Sprintf("failed to register module %q for category %q since it does not implement the BlockstoreProvider interface", module.ID(), category))
		}
	case ModuleCategoryBlockstoreWrapper:
		// module must implement the correct interface
		if _, ok := module.(BlockstoreWrapper); !ok {
			panic(fmt.Sprintf("failed to register module %q for category %q since it does not implement the BlockstoreWrapper interface", module.ID(), category))
		}
	case ModuleCategoryBootstrapper:
		// module must implement the correct interface
		if _, ok := module.(Bootstrapper); !ok {
			panic(fmt.Sprintf("failed to register module %q for category %q since it does not implement the Bootstrapper interface", module.ID(), category))
		}
	case ModuleCategoryConnManager:
		// module must implement the correct interface
		if _, ok := module.(ConnManagerProvider); !ok {
			panic(fmt.Sprintf("failed to register module %q for category %q since it does not implement the ConnManagerProvider interface", module.ID(), category))
		}
	default:
		panic(fmt.Sprintf("failed to register module %q since it specifies invalid category %q", module.ID(), category))
	}

	modulesMu.Lock()
	defer modulesMu.Unlock()

	ml := modules[category]
	if ml == nil {
		ml = map[string]Module{}
	}
	ml[module.ID()] = module
	modules[category] = ml
}

func ModulesForCategory(category string) map[string]Module {
	modulesMu.Lock()
	defer modulesMu.Unlock()

	mods, ok := modules[category]
	if !ok {
		return map[string]Module{}
	}
	return mods
}
