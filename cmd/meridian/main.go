package main

import (
	"github.com/iand/meridian"

	// standard modules
	_ "github.com/iand/meridian/modules/blockstore/basic"
	_ "github.com/iand/meridian/modules/blockstorewrapper/cache"
	_ "github.com/iand/meridian/modules/bootstrapper/basic"
	_ "github.com/iand/meridian/modules/connmanager/basic"
	_ "github.com/iand/meridian/modules/datastore/badger2"
	_ "github.com/iand/meridian/modules/datastorewrapper/log"
)

func main() {
	meridian.Main()
}
