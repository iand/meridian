package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/iand/meridian/ipfs"
)

//go:embed VERSION
var rawVersion string

var version string

func init() {
	version = rawVersion
	if idx := strings.Index(version, "\n"); idx > -1 {
		version = version[:idx]
	}
}

func main() {
	ctx := context.Background()
	if err := app.RunContext(ctx, os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

const appName = "meridian"

var app = &cli.App{
	Name:    appName,
	Usage:   "",
	Version: version,
	Flags: flagSet(
		flags,
		loggingFlags,
		diagnosticsFlags,
	),
	Before: configure,
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context
		p, err := ipfs.NewPeer(&ipfs.PeerConfig{
			ListenAddr:     config.listenAddr,
			DatastorePath:  config.datastorePath,
			FileSystemPath: config.fileSystemPath,
			Libp2pKeyFile:  config.libp2pKeyfile,
			ManifestPath:   config.manifestPath,
			Offline:        config.offline,
		})
		if err != nil {
			return fmt.Errorf("new ipfs peer: %w", err)
		}
		defer p.Close()

		gw, err := ipfs.NewGateway(p)
		if err != nil {
			return fmt.Errorf("new gateway: %w", err)
		}

		s := NewServer(gw)
		s.Serve(ctx)

		return nil
	},
}
