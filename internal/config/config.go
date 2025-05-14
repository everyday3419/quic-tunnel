package config

import (
	"crypto/tls"
	"time"

	"github.com/everyday3419/quic-tunnel/internal/util"
	"github.com/quic-go/quic-go"
)

type Config struct {
	Addr       string
	TCPAddr    string
	TLSConfig  *tls.Config
	QUICConfig *quic.Config
	Timeout    time.Duration
}

func NewDefaultConfig(addr string) *Config {
	return &Config{
		Addr:       addr,
		TCPAddr:    "localhost:8090",
		TLSConfig:  util.GenerateTLSConfig(),
		QUICConfig: &quic.Config{MaxIdleTimeout: 30 * time.Second},
		Timeout:    15 * time.Second,
	}
}
