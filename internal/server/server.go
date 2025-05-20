package server

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog"
)

type QUICConnectionHandler interface {
	handleConnection(conn quic.Connection)
}

type Server struct {
	listener *quic.Listener

	addr     string
	tlsConf  *tls.Config
	quicConf *quic.Config

	httpClient HTTPClientProvider
	quicConn   QUICConnectionHandler

	logger *zerolog.Logger
}

func New(addr string, tlsConf *tls.Config, quicConf *quic.Config, logger *zerolog.Logger) (*Server, error) {
	if quicConf == nil {
		quicConf = &quic.Config{
			MaxIdleTimeout: 30 * time.Second,
		}
	}

	listener, err := quic.ListenAddr(addr, tlsConf, quicConf)
	if err != nil {
		return nil, err
	}

	hc := newHTTPClient(logger, 15*time.Second)
	ql := newQUICConn(hc, logger)
	return &Server{
		listener:   listener,
		addr:       addr,
		tlsConf:    tlsConf,
		quicConf:   quicConf,
		httpClient: hc,
		quicConn:   ql,
		logger:     logger,
	}, nil
}

func (s *Server) Run(ctx context.Context) error {
	for {
		conn, err := s.listener.Accept(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				s.logger.Error().Err(err).Msg("failed to accept connection")
				return err
			}
		}
		s.logger.Debug().Msg("accepted new QUIC connection")
		go s.quicConn.handleConnection(conn)
	}
}

func (s *Server) Shutdown() error {
	if err := s.httpClient.close(); err != nil {
		return err
	}
	return s.listener.Close()
}
