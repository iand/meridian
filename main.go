package meridian

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"time"

	"contrib.go.opencensus.io/exporter/prometheus"
	logging "github.com/ipfs/go-log/v2"
	metricsprom "github.com/ipfs/go-metrics-prometheus"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/stats/view"

	"github.com/iand/meridian/conf"
	"github.com/iand/meridian/ipfs"
)

func Main() {
	ctx := context.Background()
	if err := app.RunContext(ctx, os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

const appName = "meridian"

func configure(_ *cli.Context) error {
	if err := logging.SetLogLevel(appName, loggingConfig.level); err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}

	if diagnosticsConfig.debugAddr != "" {
		if err := startDebugServer(); err != nil {
			return fmt.Errorf("start debug server: %w", err)
		}
	}

	if diagnosticsConfig.prometheusAddr != "" {
		if err := startPrometheusServer(); err != nil {
			return fmt.Errorf("start prometheus server: %w", err)
		}
	}

	return nil
}

func flagSet(fs ...[]cli.Flag) []cli.Flag {
	var flags []cli.Flag

	for _, f := range fs {
		flags = append(flags, f...)
	}

	return flags
}

var app = &cli.App{
	Name:    appName,
	Usage:   "",
	Version: Version,
	Flags: flagSet(
		flags,
		loggingFlags,
		diagnosticsFlags,
	),
	Before: configure,
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		configstr := `{
			"peer": {
				"key_file": "/mnt/disk1/data/meridian/peer.key",
				"listen_addrs": ["/ip4/0.0.0.0/tcp/4005"],
				"offline": false
			},
			"datastore": {
				"badger": {
					"path": "/mnt/disk1/data/meridian/store"
				}
			},
			"blockstore": {
				"basic": {}
			},
			"blockstore_wrapper": {
				"cache": {}
			},
			"routing": {
				"fullrt": {
					"bootstrap_peers": [
						"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
						"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
						"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
						"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt"
					]
				}
			},
			"bootstrapper": {
				"basic": {
					"bootstrap_peers": [
						"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
						"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
						"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
						"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt"
					]
				}
			}
		}`

		var cfg conf.Config
		if err := json.Unmarshal([]byte(configstr), &cfg); err != nil {
			return fmt.Errorf("read config: %w", err)
		}

		p, err := ipfs.NewPeer(&cfg)
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

var (
	config struct {
		fileSystemPath            string
		offline                   bool
		garbageCollectionInterval time.Duration
	}

	flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "fileroot",
			Aliases:     []string{"ipfs-fileroot"}, // old name for this flag
			Usage:       "Path to root of filesystem to be served.",
			Value:       "/mnt/disk1/data/meridian/files", // TODO: remove default
			Destination: &config.fileSystemPath,
			EnvVars:     []string{"meridian_FILEROOT"},
		},
		&cli.BoolFlag{
			Name:        "offline",
			Usage:       "When true, don't connect to the public ipfs dht.",
			Value:       false,
			Destination: &config.offline,
			EnvVars:     []string{"meridian_OFFLINE"},
		},
		&cli.DurationFlag{
			Name:        "gc-interval",
			Usage:       "Controls how frequently the ipfs blockstore should be garbage collected to remove orphaned blocks.",
			Value:       24 * time.Hour,
			Destination: &config.garbageCollectionInterval,
			EnvVars:     []string{"meridian_GC_INTERVAL"},
		},
	}
)

var (
	logger = logging.Logger(appName)

	loggingConfig struct {
		level string
	}

	loggingFlags = []cli.Flag{
		&cli.StringFlag{
			Name:        "log-level",
			EnvVars:     []string{"meridian_LOG_LEVEL"},
			Value:       "DEBUG",
			Usage:       "Set the default log level for the " + appName + " logger to `LEVEL`",
			Destination: &loggingConfig.level,
		},
	}
)

var (
	diagnosticsConfig struct {
		debugAddr      string
		prometheusAddr string
	}

	diagnosticsFlags = []cli.Flag{
		&cli.StringFlag{
			Name:        "debug-addr",
			Usage:       "Network address to start a debug http server on (example: 127.0.0.1:8080)",
			Value:       "",
			Destination: &diagnosticsConfig.debugAddr,
			EnvVars:     []string{"meridian_DEBUG_ADDR"},
		},
		&cli.StringFlag{
			Name:        "prometheus-addr",
			Usage:       "Network address to start a prometheus metric exporter server on (example: :9991)",
			Value:       "",
			Destination: &diagnosticsConfig.prometheusAddr,
			EnvVars:     []string{"meridian_PROMETHEUS_ADDR"},
		},
	}
)

const (
	DefaultFirstBlockSize = 4096    // Ensure the first block in the dag is small to optimize listing directory sizes (by go-ipfs, for example)
	DefaultBlockSize      = 1 << 20 // 1MiB
)

func startPrometheusServer() error {
	// Bind the ipfs metrics interface to prometheus
	if err := metricsprom.Inject(); err != nil {
		logger.Errorw("unable to inject prometheus ipfs/go-metrics exporter; some metrics will be unavailable", "error", err)
	}

	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace:  appName,
		Registerer: prom.DefaultRegisterer,
		Gatherer:   prom.DefaultGatherer,
	})
	if err != nil {
		return fmt.Errorf("new prometheus exporter: %w", err)
	}

	// register prometheus with opencensus
	view.RegisterExporter(pe)
	view.SetReportingPeriod(2 * time.Second)

	mux := http.NewServeMux()
	mux.Handle("/metrics", pe)
	go func() {
		if err := http.ListenAndServe(diagnosticsConfig.prometheusAddr, mux); err != nil {
			logger.Errorw("prometheus server failed", "error", err)
		}
	}()
	return nil
}

func startDebugServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))

	go func() {
		if err := http.ListenAndServe(diagnosticsConfig.debugAddr, mux); err != nil {
			logger.Errorw("debug server failed", "error", err)
		}
	}()
	return nil
}
