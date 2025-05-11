package main

import (
	"crypto/tls"
	"os"

	"github.com/everyday3419/quic-tunnel/internal/app"
	"github.com/rs/zerolog"
)

const addr = "localhost:4242"

func main() {
	output := zerolog.ConsoleWriter{Out: os.Stdout, NoColor: false}
	logger := zerolog.New(output).With().Timestamp().Logger()

	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-tunnel"},
	}

	tunnel := app.NewClient(addr, logger, tlsConf)
	tunnel.Run()
}
