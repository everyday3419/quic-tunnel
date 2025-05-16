package server

import (
	"context"
	"net/http"

	"github.com/everyday3419/quic-tunnel/internal/config"
	"github.com/everyday3419/quic-tunnel/internal/forwarder"
	"github.com/everyday3419/quic-tunnel/internal/processor"
	"github.com/quic-go/quic-go"

	"github.com/everyday3419/quic-tunnel/internal/transport"
	"github.com/rs/zerolog"
)

type TransportProvider interface {
	Serve(ctx context.Context, handler transport.StreamHandler) error
	Close() error
}

type ProcessorProvider interface {
	ProcessStream(stream quic.Stream, handler processor.RequestHandler) error
}

type ForwarderProvider interface {
	HandleRequest(req *http.Request) (*http.Response, error)
	Close() error
}

type Server struct {
	transport TransportProvider
	processor ProcessorProvider
	forwarder ForwarderProvider
	logger    *zerolog.Logger
}

func New(cfg *config.Config, logger *zerolog.Logger) (*Server, error) {
	transport, err := transport.NewQUICTransport(cfg.Addr, cfg.TLSConfig, cfg.QUICConfig, logger)
	if err != nil {
		return nil, err
	}
	processor := processor.New(logger)
	forwarder := forwarder.New(logger, cfg.Timeout)
	return &Server{
		transport: transport,
		processor: processor,
		forwarder: forwarder,
		logger:    logger,
	}, nil
}

func (s *Server) Run(ctx context.Context) error {
	return s.transport.Serve(ctx, func(stream quic.Stream) error {
		return s.processor.ProcessStream(stream, s.forwarder.HandleRequest)
	})
}

func (s *Server) Shutdown() error {
	if err := s.transport.Close(); err != nil {
		return err
	}
	return s.forwarder.Close()
}
