package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/iand/meridian/ipfs"
	"github.com/ipfs/go-ipfs/core/corehttp"
)

type Server struct {
	Addr    string
	Net     string
	Gateway *ipfs.Gateway
	mux     *http.ServeMux
}

// New creates a new server with default values.
func NewServer(gw *ipfs.Gateway) *Server {
	return &Server{
		Addr:    ":2525",
		Net:     "tcp",
		Gateway: gw,
	}
}

// Serve starts the server and blocks until the process receives a terminating operating system signal.
func (s *Server) Serve(ctx context.Context) error {
	if s.Addr == "" {
		s.Addr = ":2525"
	}

	if s.Net == "" {
		s.Net = "tcp"
	}

	listener, err := net.Listen(s.Net, s.Addr)
	if err != nil {
		return fmt.Errorf("fatal error listening on %s: %w", s.Addr, err)
	}

	if err := s.registerHandlers(); err != nil {
		return fmt.Errorf("fatal error registering handlers: %w", err)
	}

	hs := &http.Server{
		Addr:    s.Addr,
		Handler: s,
	}

	go hs.Serve(listener)

	s.waitSignal()
	return nil
}

// waitSignal blocks waiting for operating system signals
func (s *Server) waitSignal() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

signalloop:
	for sig := range ch {
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			break signalloop
			// TODO: support HUP to reload config
		}
	}
}

func (s *Server) registerHandlers() error {
	s.mux = http.NewServeMux()

	headers := make(map[string][]string)
	corehttp.AddAccessControlHeaders(headers)

	cfg := corehttp.GatewayConfig{
		Writable: false,
		Headers:  headers,
	}

	gwHandler := corehttp.NewGatewayHandler(cfg, s.Gateway)
	s.mux.Handle("/ipfs/", gwHandler)
	s.mux.Handle("/ipns/", gwHandler)

	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
