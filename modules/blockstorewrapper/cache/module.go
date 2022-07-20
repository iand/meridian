package cache

import (
	"context"
	"fmt"

	"github.com/iand/meridian/conf"
	"github.com/ipfs/go-ipfs-blockstore"
)

func init() {
	defaults := blockstore.DefaultCacheOpts()

	conf.RegisterModule(conf.ModuleCategoryBlockstoreWrapper, &Cache{
		BloomFilterSize:   defaults.HasBloomFilterSize,
		BloomFilterHashes: defaults.HasBloomFilterHashes,
		ARCCacheSize:      defaults.HasARCCacheSize,
	})
}

type Cache struct {
	BloomFilterSize   int `json:"bloom_filter_size"`
	BloomFilterHashes int `json:"bloom_filter_hashes"`
	ARCCacheSize      int `json:"arc_cache_size"`
}

func (b *Cache) ID() string { return "cache" }

func (b *Cache) ValidateModule() error {
	return nil
}

func (b *Cache) WrapBlockstore(ctx context.Context, bs blockstore.Blockstore) (blockstore.Blockstore, error) {
	cbs, err := blockstore.CachedBlockstore(ctx, bs, blockstore.DefaultCacheOpts())
	if err != nil {
		return nil, fmt.Errorf("new cached blockstore: %w", err)
	}
	return cbs, nil
}
