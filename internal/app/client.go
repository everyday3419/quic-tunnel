package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog"
)

type Client struct {
	addr    string
	logger  zerolog.Logger
	tlsConf *tls.Config
}

func NewClient(
	addr string,
	logger zerolog.Logger,
	tlsConf *tls.Config,
) *Client {
	return &Client{
		addr:    addr,
		logger:  logger,
		tlsConf: tlsConf,
	}
}

func (t *Client) Run() {
	conn, err := quic.DialAddr(context.Background(), t.addr, t.tlsConf, nil)
	if err != nil {
		t.logger.Err(err).Msg("")
	}

	stream, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		t.logger.Err(err).Msg("")
	}
	defer stream.Close()

	for {
		req, err := http.NewRequest("GET", "https://cloudflare.com", nil)
		if err != nil {
			t.logger.Err(err).Msg("")
			stream.Close()
			continue
		}

		fmt.Printf("client: sending '%s'\n", req)
		if err := req.Write(stream); err != nil {
			t.logger.Err(err).Msg("")
			stream.Close()
			continue
		}

		buf := make([]byte, 4096)
		n, err := stream.Read(buf)
		if err != nil && err != io.EOF {
			t.logger.Error().Err(err).Msg("failed to read from stream")
			return
		}

		fmt.Printf("client: got '%s'\n", buf[:n])

		time.Sleep(3 * time.Second)
	}
}
