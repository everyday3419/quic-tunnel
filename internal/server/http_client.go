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

func (h *httpClient) handleRequest(req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(req.Context(), h.timeout)
	defer cancel()

	// Try HTTP/3 first
	resp, err := h.http3Client.Do(req.WithContext(ctx))
	if err != nil {
		h.logger.Warn().Err(err).Msg("HTTP/3 not supported, falling back to HTTP/2 or HTTP/1.1")
		// Fallback to HTTP/2 or HTTP/1.1
		return h.http2Client.Do(req.WithContext(ctx))
	}
	return resp, nil
}

func (h *httpClient) close() error {
	h.http3Client.Transport.(*http3.Transport).Close()
	return nil
}
