package app

import (
	"context"

	"github.com/everyday3419/quic-tunnel/internal/config"
	"github.com/everyday3419/quic-tunnel/internal/http"
	"github.com/everyday3419/quic-tunnel/internal/transport"
	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog"
)

type Server struct {
	transport *transport.QUICTransport
	httpProc  *http.HTTPProcessor
	proxy     *http.Proxy
	logger    *zerolog.Logger
}

func NewServer(cfg *config.Config, logger *zerolog.Logger) (*Server, error) {
	transport, err := transport.NewQUICTransport(cfg.Addr, cfg.TLSConfig, cfg.QUICConfig, logger)
	if err != nil {
		return nil, err
	}
	httpProc := http.NewHTTPProcessor(logger)
	proxy := http.NewProxy(logger, cfg.Timeout)
	return &Server{
		transport: transport,
		httpProc:  httpProc,
		proxy:     proxy,
		logger:    logger,
	}, nil
}

func (s *Server) Serve(ctx context.Context) error {
	return s.transport.Serve(ctx, func(stream quic.Stream) error {
		return s.httpProc.ProcessStream(stream, s.proxy.HandleRequest)
	})
}

func (s *Server) Shutdown() error {
	if err := s.transport.Close(); err != nil {
		return err
	}
	return s.proxy.Close()
}
