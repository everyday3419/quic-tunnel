package transport

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog"
)

type StreamHandler func(quic.Stream) error

type QUICTransport struct {
	listener *quic.Listener
	config   *quic.Config
	logger   *zerolog.Logger
}

func NewQUICTransport(addr string, tlsConfig *tls.Config, quicConfig *quic.Config, logger *zerolog.Logger) (*QUICTransport, error) {
	if quicConfig == nil {
		quicConfig = &quic.Config{
			MaxIdleTimeout: 30 * time.Second,
		}
	}
	listener, err := quic.ListenAddr(addr, tlsConfig, quicConfig)
	if err != nil {
		return nil, err
	}
	return &QUICTransport{
		listener: listener,
		config:   quicConfig,
		logger:   logger,
	}, nil
}

func (t *QUICTransport) Serve(ctx context.Context, handler StreamHandler) error {
	for {
		conn, err := t.listener.Accept(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				t.logger.Error().Err(err).Msg("failed to accept connection")
				return err
			}
		}
		t.logger.Debug().Msg("accepted new QUIC connection")
		go t.handleConnection(conn, handler)
	}
}

func (t *QUICTransport) handleConnection(conn quic.Connection, handler StreamHandler) {
	for {
		stream, err := conn.AcceptStream(context.Background())
		if err != nil {
			t.logger.Error().Err(err).Msg("failed to accept stream")
			return
		}
		go func() {
			if err := handler(stream); err != nil {
				// t.logger.Error().Err(err).Msg("failed to handle stream")
			}
		}()
	}
}

func (t *QUICTransport) Close() error {
	return t.listener.Close()
}
