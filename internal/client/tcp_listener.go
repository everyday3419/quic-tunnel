package client

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

type QUICForwarder interface {
	forwardToQUIC(ctx context.Context, req *http.Request) (*http.Response, error)
}

type tcpListener struct {
	forwardToQUIC QUICForwarder

	logger zerolog.Logger
}

func newTCPListener(forwardToQUIC QUICForwarder, logger *zerolog.Logger) *tcpListener {
	return &tcpListener{
		forwardToQUIC: forwardToQUIC,
		logger:        *logger,
	}
}

func (tl *tcpListener) handleTCPConnection(ctx context.Context, tcpConn net.Conn) {
	defer tcpConn.Close()

	if err := tcpConn.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
		tl.logger.Error().Err(err).Msg("failed to set TCP read deadline")
		return
	}

	buf := make([]byte, 4096)
	n, err := tcpConn.Read(buf)
	if err != nil && err != io.EOF {
		tl.logger.Error().Err(err).Msg("failed to read from TCP")
		return
	}

	reader := bufio.NewReader(bytes.NewReader(buf[:n]))
	req, err := http.ReadRequest(reader)
	if err != nil {
		tl.logger.Error().Err(err).Msg("failed to parse HTTP request")
		return
	}

	req.URL.Scheme = "https"
	if req.Host == "" {
		req.Host = req.URL.Host
	}
	req.RequestURI = ""

	resp, err := tl.forwardToQUIC.forwardToQUIC(ctx, req)
	if err != nil {
		tl.logger.Error().Err(err).Msg("failed to forward request to QUIC")
		return
	}

	if err := tcpConn.SetWriteDeadline(time.Now().Add(15 * time.Second)); err != nil {
		tl.logger.Error().Err(err).Msg("failed to set TCP write deadline")
		return
	}

	if err := resp.Write(tcpConn); err != nil {
		tl.logger.Error().Err(err).Msg("failed to write response to TCP")
		return
	}

	tl.logger.Info().Msgf("forwarded response to TCP: %s", resp.Status)
}
