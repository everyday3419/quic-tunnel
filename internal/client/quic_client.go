package client

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog"
)

type quicClient struct {
	serverAddr string
	tlsConf    *tls.Config
	quicConf   *quic.Config

	logger *zerolog.Logger
}

func newQUICClient(serverAddr string, tlsConf *tls.Config, quicConf *quic.Config, logger *zerolog.Logger) *quicClient {
	return &quicClient{
		serverAddr: serverAddr,
		tlsConf:    tlsConf,
		quicConf:   quicConf,
		logger:     logger,
	}
}

func (qc *quicClient) forwardToQUIC(ctx context.Context, req *http.Request) (*http.Response, error) {
	quicConn, err := quic.DialAddr(ctx, qc.serverAddr, qc.tlsConf, qc.quicConf)
	if err != nil {
		qc.logger.Error().Err(err).Msg("failed to dial QUIC server")
		return nil, err
	}
	defer quicConn.CloseWithError(0, "")

	stream, err := quicConn.OpenStreamSync(ctx)
	if err != nil {
		qc.logger.Error().Err(err).Msg("failed to open QUIC stream")
		return nil, err
	}
	defer stream.Close()

	if err := stream.SetWriteDeadline(time.Now().Add(15 * time.Second)); err != nil {
		qc.logger.Error().Err(err).Msg("failed to set QUIC write deadline")
		return nil, err
	}

	qc.logger.Info().Msgf("forwarding request to %s via QUIC", req.URL)
	if err := req.Write(stream); err != nil {
		qc.logger.Error().Err(err).Msg("failed to write request to QUIC stream")
		return nil, err
	}

	if err := stream.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
		qc.logger.Error().Err(err).Msg("failed to set QUIC read deadline")
		return nil, err
	}

	// unexpected EOF
	respBuf := make([]byte, 4096)
	n, err := stream.Read(respBuf)
	if err != nil && err != io.EOF {
		qc.logger.Error().Err(err).Msg("failed to read from QUIC stream")
		return nil, err
	}

	respReader := bufio.NewReader(bytes.NewReader(respBuf[:n]))
	resp, err := http.ReadResponse(respReader, req)
	if err != nil && err != io.EOF {
		qc.logger.Error().Err(err).Msg("failed to parse HTTP response")
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil && err != io.EOF {
		qc.logger.Error().Err(err).Msg("failed to read response body for logging")
		return nil, err
	}

	qc.logger.Info().
		Str("status", resp.Status).
		Interface("headers", resp.Header).
		Str("body", string(bodyBytes)).
		Msg("received HTTP response from QUIC")

	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	return resp, nil
}
