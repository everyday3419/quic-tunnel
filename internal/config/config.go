package config

import (
	"crypto/tls"
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/everyday3419/quic-tunnel/internal/util"
	"github.com/quic-go/quic-go"
)

type Config interface {
	*ClientConfig | *ServerConfig
}

type ConfigOpts interface {
	ClientConfigOpts | ServerConfigOpts
}

type ClientConfig struct {
	ClientConfigOpts

	TLSConf    *tls.Config
	QUICConfig *quic.Config
}

type ClientConfigOpts struct {
	ListenAddr string
	ServerAddr string
	Timeout    time.Duration
}

type ServerConfig struct {
	ServerConfigOpts

	TLSConfig  *tls.Config
	QUICConfig *quic.Config
}

type ServerConfigOpts struct {
	Addr string
}

func New[T Config, O ConfigOpts](path string) (T, error) {
	opts, err := loadConf[O](path)
	if err != nil {
		return nil, err
	}

	switch any(*new(T)).(type) {
	case *ServerConfig:
		serverOpts, ok := any(*opts).(ServerConfigOpts)
		if !ok {
			return nil, fmt.Errorf("invalid opts type for ServerConfig: %T", opts)
		}
		return any(&ServerConfig{
			ServerConfigOpts: serverOpts,
			TLSConfig:        util.GenerateTLSConfig(),
			QUICConfig:       &quic.Config{MaxIdleTimeout: 30 * time.Second},
		}).(T), nil
	case *ClientConfig:
		clientOpts, ok := any(*opts).(ClientConfigOpts)
		if !ok {
			return nil, fmt.Errorf("invalid opts type for ClientConfig: %T", opts)
		}
		return any(&ClientConfig{
			ClientConfigOpts: clientOpts,
			TLSConf: &tls.Config{
				InsecureSkipVerify: true,
				NextProtos:         []string{"quic-tunnel"},
			},
			QUICConfig: &quic.Config{MaxIdleTimeout: 30 * time.Second},
		}).(T), nil
	default:
		return nil, fmt.Errorf("unsupported config type")
	}
}

func loadConf[T ConfigOpts](path string) (*T, error) {
	if path == "" {
		return nil, fmt.Errorf("path is empty")
	}
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	var conf T
	_, err := toml.DecodeFile(path, &conf)
	if err != nil {
		return nil, err
	}
	return &conf, nil
}
