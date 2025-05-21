package server

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog"
)

type HTTPClientProvider interface {
	handleRequest(ctx context.Context, req *http.Request) (*http.Response, error)
	close() error
}

type quicConn struct {
	httpClient HTTPClientProvider

	logger *zerolog.Logger
}

func newQUICConn(httpClient HTTPClientProvider, logger *zerolog.Logger) *quicConn {
	return &quicConn{
		httpClient: httpClient,
		logger:     logger,
	}
}

func (ql *quicConn) handleConnection(ctx context.Context, conn quic.Connection) {
	for {
		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			ql.logger.Error().Err(err).Msg("failed to accept stream")
			return
		}
		go func() {
			if err := ql.handleStream(ctx, stream); err != nil {
				// t.logger.Error().Err(err).Msg("failed to handle stream")
			}
		}()
	}

}

func (ql *quicConn) handleStream(ctx context.Context, stream quic.Stream) error {
	if err := stream.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return err
	}

	buf := make([]byte, 4096)
	n, err := stream.Read(buf)
	if err != nil && err != io.EOF {
		return err
	}

	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(buf[:n])))
	if err != nil {
		return err
	}

	req.URL.Scheme = "https"
	req.URL.Host = req.Host
	req.RequestURI = ""

	resp, err := ql.httpClient.handleRequest(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	ql.logger.Info().
		Str("protocol", resp.Proto).
		Int("status", resp.StatusCode).
		Msg("received HTTP response")

	if err := stream.SetWriteDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return err
	}
	return resp.Write(stream)
}
