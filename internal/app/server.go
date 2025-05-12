package app

import (
	"context"
	"net/http"

	"github.com/everyday3419/quic-tunnel/internal/config"
	h "github.com/everyday3419/quic-tunnel/internal/http"
	"github.com/everyday3419/quic-tunnel/internal/proxy"
	"github.com/everyday3419/quic-tunnel/internal/transport"
	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog"
)

type TransportProvider interface {
	Serve(ctx context.Context, handler transport.StreamHandler) error
	Close() error
}

type HTTPProvider interface {
	ProcessStream(stream quic.Stream, handler h.RequestHandler) error
}

type ProxyProvider interface {
	HandleRequest(req *http.Request) (*http.Response, error)
	Close() error
}

type Server struct {
	transport TransportProvider
	httpProc  HTTPProvider
	proxy     ProxyProvider
	logger    *zerolog.Logger
}

func NewServer(cfg *config.Config, logger *zerolog.Logger) (*Server, error) {
	transport, err := transport.NewQUICTransport(cfg.Addr, cfg.TLSConfig, cfg.QUICConfig, logger)
	if err != nil {
		return nil, err
	}
	httpProc := h.NewHTTPProcessor(logger)
	proxy := proxy.NewProxy(logger, cfg.Timeout)
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
