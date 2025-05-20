package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/everyday3419/quic-tunnel/internal/client"
	"github.com/everyday3419/quic-tunnel/internal/config"
	"github.com/rs/zerolog"
)

func main() {
	conf, err := config.New[*config.ClientConfig, config.ClientConfigOpts]("conf/client.toml")
	if err != nil {
		log.Fatal(err)
	}

	output := zerolog.ConsoleWriter{Out: os.Stdout, NoColor: false}
	logger := zerolog.New(output).With().Timestamp().Logger()

	c := client.New(conf.ListenAddr, conf.ServerAddr, conf.TLSConf, nil, &logger)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	go func() {
		if err := c.Run(ctx); err != nil {
			logger.Error().Err(err).Msg("client failed")
		}
	}()

	<-ctx.Done()
}
