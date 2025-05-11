package http

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/rs/zerolog"
)

type Proxy struct {
	http3Client *http.Client
	http2Client *http.Client
	logger      *zerolog.Logger
	timeout     time.Duration
}

func NewProxy(logger *zerolog.Logger, timeout time.Duration) *Proxy {
	http3Transport := &http3.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		QUICConfig:      &quic.Config{},
	}
	http2Transport := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"h2", "http/1.1"}},
		ForceAttemptHTTP2: true,
	}
	return &Proxy{
		http3Client: &http.Client{Transport: http3Transport},
		http2Client: &http.Client{Transport: http2Transport},
		logger:      logger,
		timeout:     timeout,
	}
}

func (p *Proxy) HandleRequest(req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(req.Context(), p.timeout)
	defer cancel()

	// Try HTTP/3 first
	resp, err := p.http3Client.Do(req.WithContext(ctx))
	if err != nil {
		p.logger.Warn().Err(err).Msg("HTTP/3 not supported, falling back to HTTP/2 or HTTP/1.1")
		// Fallback to HTTP/2 or HTTP/1.1
		return p.http2Client.Do(req.WithContext(ctx))
	}
	return resp, nil
}

func (p *Proxy) Close() error {
	p.http3Client.Transport.(*http3.Transport).Close()
	return nil
}
