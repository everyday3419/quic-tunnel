package server

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/rs/zerolog"
)

type httpClient struct {
	http3Client *http.Client
	http2Client *http.Client

	timeout time.Duration

	logger *zerolog.Logger
}

func newHTTPClient(logger *zerolog.Logger, timeout time.Duration) *httpClient {
	http3Transport := &http3.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		QUICConfig:      &quic.Config{},
	}
	http2Transport := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"h2", "http/1.1"}},
		ForceAttemptHTTP2: true,
	}
	return &httpClient{
		http3Client: &http.Client{Transport: http3Transport},
		http2Client: &http.Client{Transport: http2Transport},
		timeout:     timeout,
		logger:      logger,
	}
}

func (h *httpClient) handleRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	req = req.WithContext(ctx)

	resp, err := h.http3Client.Do(req)
	if err == nil {
		h.logger.Info().
			Str("protocol", "HTTP/3").
			Str("url", req.URL.String()).
			Int("status", resp.StatusCode).
			Msg("successfully sent request")
		return resp, nil
	}

	if ctx.Err() != nil {
		h.logger.Warn().
			Err(ctx.Err()).
			Str("url", req.URL.String()).
			Msg("request cancelled or timed out")
		return nil, ctx.Err()
	}

	h.logger.Warn().
		Err(err).
		Str("url", req.URL.String()).
		Msg("HTTP/3 not supported, falling back to HTTP/2 or HTTP/1.1")

	resp, err = h.http2Client.Do(req)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("url", req.URL.String()).
			Msg("failed to send request with HTTP/2 or HTTP/1.1")
		return nil, err
	}

	h.logger.Info().
		Str("protocol", resp.Proto).
		Str("url", req.URL.String()).
		Int("status", resp.StatusCode).
		Msg("successfully sent request")
	return resp, nil
}

func (h *httpClient) close() error {
	return h.http3Client.Transport.(*http3.Transport).Close()
}
