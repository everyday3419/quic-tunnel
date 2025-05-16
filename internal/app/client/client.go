package client

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog"
)

type Client struct {
	addr       string
	serverAddr string
	logger     zerolog.Logger
	tlsConf    *tls.Config
	quicConf   *quic.Config
}

func New(addr, serverAddr string, logger zerolog.Logger, tlsConf *tls.Config, quicConf *quic.Config) *Client {
	if quicConf == nil {
		quicConf = &quic.Config{
			MaxIdleTimeout: 30 * time.Second,
		}
	}
	return &Client{
		addr:       addr,
		serverAddr: serverAddr,
		logger:     logger,
		tlsConf:    tlsConf,
		quicConf:   quicConf,
	}
}

func (c *Client) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", c.addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", c.addr, err)
	}
	defer listener.Close()

	c.logger.Info().Msgf("listening for TCP connections on %s", c.addr)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			conn, err := listener.Accept()
			if err != nil {
				c.logger.Error().Err(err).Msg("failed to accept TCP connection")
				continue
			}
			c.logger.Debug().Msgf("accepted TCP connection from %s", conn.RemoteAddr().String())

			go c.handleTCPConnection(ctx, conn)
		}
	}
}

func (c *Client) handleTCPConnection(ctx context.Context, tcpConn net.Conn) {
	defer tcpConn.Close()

	if err := tcpConn.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
		c.logger.Error().Err(err).Msg("failed to set TCP read deadline")
		return
	}

	buf := make([]byte, 4096)
	n, err := tcpConn.Read(buf)
	if err != nil && err != io.EOF {
		c.logger.Error().Err(err).Msg("failed to read from TCP")
		return
	}

	reader := bufio.NewReader(bytes.NewReader(buf[:n]))
	req, err := http.ReadRequest(reader)
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to parse HTTP request")
		return
	}

	req.URL.Scheme = "https"
	if req.Host == "" {
		req.Host = req.URL.Host
	}
	req.RequestURI = ""

	quicConn, err := quic.DialAddr(ctx, c.serverAddr, c.tlsConf, c.quicConf)
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to dial QUIC server")
		return
	}
	defer quicConn.CloseWithError(0, "")

	stream, err := quicConn.OpenStreamSync(ctx)
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to open QUIC stream")
		return
	}
	defer stream.Close()

	if err := stream.SetWriteDeadline(time.Now().Add(15 * time.Second)); err != nil {
		c.logger.Error().Err(err).Msg("failed to set QUIC write deadline")
		return
	}

	c.logger.Info().Msgf("forwarding request to %s via QUIC", req.URL)
	if err := req.Write(stream); err != nil {
		c.logger.Error().Err(err).Msg("failed to write request to QUIC stream")
		return
	}

	if err := stream.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
		c.logger.Error().Err(err).Msg("failed to set QUIC read deadline")
		return
	}

	// unexpected EOF
	respBuf := make([]byte, 4096)
	n, err = stream.Read(respBuf)
	if err != nil && err != io.EOF {
		c.logger.Error().Err(err).Msg("failed to read from QUIC stream")
		return
	}

	respReader := bufio.NewReader(bytes.NewReader(respBuf[:n]))
	resp, err := http.ReadResponse(respReader, req)
	if err != nil && err != io.EOF {
		c.logger.Error().Err(err).Msg("failed to parse HTTP response")
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil && err != io.EOF {
		c.logger.Error().Err(err).Msg("failed to read response body for logging")
		return
	}

	c.logger.Info().
		Str("status", resp.Status).
		Interface("headers", resp.Header).
		Str("body", string(bodyBytes)).
		Msg("received HTTP response from QUIC")

	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	if err := tcpConn.SetWriteDeadline(time.Now().Add(15 * time.Second)); err != nil {
		c.logger.Error().Err(err).Msg("failed to set TCP write deadline")
		return
	}

	if err := resp.Write(tcpConn); err != nil {
		c.logger.Error().Err(err).Msg("failed to write response to TCP")
		return
	}

	c.logger.Info().Msgf("forwarded response to TCP: %s", resp.Status)
}
