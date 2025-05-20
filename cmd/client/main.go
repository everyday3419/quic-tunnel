package main

import (
	"context"
	"crypto/tls"
	"os"
	"os/signal"

	"github.com/everyday3419/quic-tunnel/internal/client"
	"github.com/rs/zerolog"
)

const listenAddr = "localhost:8888"
const serverAddr = "localhost:4242"

func main() {
	output := zerolog.ConsoleWriter{Out: os.Stdout, NoColor: false}
	logger := zerolog.New(output).With().Timestamp().Logger()

	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-tunnel"},
	}

	c := client.New(listenAddr, serverAddr, tlsConf, nil, &logger)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	go func() {
		if err := c.Run(ctx); err != nil {
			logger.Error().Err(err).Msg("client failed")
		}
	}()

	<-ctx.Done()
}
