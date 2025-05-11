package config

import (
	"crypto/tls"
	"time"

	t "github.com/everyday3419/quic-tunnel/internal/tls"
	"github.com/quic-go/quic-go"
)

type Config struct {
	Addr       string
	TLSConfig  *tls.Config
	QUICConfig *quic.Config
	Timeout    time.Duration
}

func NewDefaultConfig(addr string) *Config {
	return &Config{
		Addr:       addr,
		TLSConfig:  t.GenerateTLSConfig(),
		QUICConfig: &quic.Config{MaxIdleTimeout: 30 * time.Second},
		Timeout:    15 * time.Second,
	}
}
