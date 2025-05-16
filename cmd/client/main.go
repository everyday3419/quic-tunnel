package main

import (
	"context"
	"crypto/tls"
	"os"

	"github.com/everyday3419/quic-tunnel/internal/app/client"
	"github.com/rs/zerolog"
)

const addr = "localhost:8888"
const serverAddr = "localhost:4242"

func main() {
	output := zerolog.ConsoleWriter{Out: os.Stdout, NoColor: false}
	logger := zerolog.New(output).With().Timestamp().Logger()

	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-tunnel"},
	}

	tunnel := client.New(addr, serverAddr, logger, tlsConf, nil)
	tunnel.Run(context.Background())
}
