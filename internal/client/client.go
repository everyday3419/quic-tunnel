package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog"
)

type TCPConnectionHandler interface {
	HandleTCPConnection(ctx context.Context, tcpConn net.Conn)
}

type Client struct {
	listenAddr string
	serverAddr string
	tlsConf    *tls.Config
	quicConf   *quic.Config

	tcpListener TCPConnectionHandler

	logger *zerolog.Logger
}

func New(listenAddr string, serverAddr string, tlsConf *tls.Config, quicConf *quic.Config, logger *zerolog.Logger) *Client {
	if quicConf == nil {
		quicConf = &quic.Config{
			MaxIdleTimeout: 30 * time.Second,
		}
	}

	qc := newQUICClient(serverAddr, tlsConf, quicConf, logger)
	tl := newTCPListener(qc, logger)

	return &Client{
		listenAddr:  listenAddr,
		serverAddr:  serverAddr,
		tlsConf:     tlsConf,
		quicConf:    quicConf,
		tcpListener: tl,
		logger:      logger,
	}
}

func (c *Client) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", c.listenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", c.listenAddr, err)
	}
	defer listener.Close()

	c.logger.Info().Msgf("listening for TCP connections on %s", c.listenAddr)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			conn, err := listener.Accept()
			if err != nil {
				c.logger.Error().Err(err).Msg("failed to accept TCP connection")
				continue
			}
			c.logger.Debug().Msgf("accepted TCP connection from %s", conn.RemoteAddr().String())
			go c.tcpListener.HandleTCPConnection(ctx, conn)
		}
	}
}
