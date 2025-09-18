package main

import (
	"fmt"
	"os"
	"time"

	"github.com/bloXroute-Labs/ton-trader-api-client/pkg/ttac"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

var (
	bts, rev, version string
)

func main() {
	zerolog.TimestampFieldName = "t"
	zerolog.LevelFieldName = "l"
	zerolog.MessageFieldName = "m"
	zerolog.TimeFieldFormat = "2006-01-02T15:04:05.000000"

	version = fmt.Sprintf("ttc::%s::%s", bts, rev)
	log.Info().Msgf("version = %s", version)
	app := &cli.App{
		Name:  "TON trader API client",
		Usage: "make requests to ton-trader-api service",
		Action: func(cc *cli.Context) error {
			_, err := ttac.Run(cc)
			return err
		},
		Flags: []cli.Flag{
			&cli.Int64Flag{
				Name:    ttac.ArgAmount,
				Aliases: []string{"a"},
				Value:   250000000,
				Usage:   "amount, default: 0.25 TON",
			},
			&cli.StringFlag{
				Name:     ttac.ArgAuthHeader,
				Aliases:  []string{"ah"},
				Required: true,
				Usage:    "bloXroute auth header",
			},
			&cli.StringFlag{
				Name:    ttac.ArgComment,
				Aliases: []string{"c"},
				Value:   fmt.Sprintf("TON trader API test, %s", time.Now().UTC().Format(zerolog.TimeFieldFormat)),
				Usage:   "transfer comment",
			},
			&cli.StringFlag{
				Name:    ttac.ArgDestinationAddress,
				Aliases: []string{"tda"},
				Usage:   "transaction destination address",
			},
			&cli.StringFlag{
				Name:  ttac.ArgEndPointURI,
				Value: "https://frankfurt.ton.dex.blxrbdn.com",
				Usage: "TON trader API endpoint",
			},
			&cli.StringFlag{
				Name:    ttac.ArgLogLevel,
				Aliases: []string{"ll"},
				Value:   "info",
				Usage:   "log `level`, one of: debug, info, warn, error",
			},
			&cli.Int64Flag{
				Name:    ttac.ArgRandomAddon,
				Aliases: []string{"ra"},
				Value:   2500000,
				Usage:   "random `addon` to the specified amount, default: 0.0025 TON",
			},
			&cli.UintFlag{
				Name:    ttac.ArgRandomPause,
				Aliases: []string{"rp"},
				Usage:   "random `pause` to take before sending, in seconds",
			},
			&cli.Int64Flag{
				Name:    ttac.ArgTip,
				Aliases: []string{"t"},
				Value:   15000000,
				Usage:   "`tip`, default: 0.015 TON",
			},
			&cli.StringFlag{
				Name:    ttac.ArgTonRPCURI,
				Aliases: []string{"rpc"},
				Value:   "https://ton.org/global-config.json",
				Usage:   "TON RPC configuration to use",
			},
			&cli.StringFlag{
				Name:     ttac.ArgWallet1,
				Aliases:  []string{"w1"},
				Required: true,
				Usage:    "file `path` with the seed phrase for (sending?) wallet",
			},
			&cli.StringFlag{
				Name:    ttac.ArgWallet2,
				Aliases: []string{"w2"},
				Usage:   "file `path` with the seed phrase for second wallet",
			},
			&cli.StringFlag{
				Name:    ttac.ArgWalletType,
				Aliases: []string{"wt"},
				Value:   "V4R2",
				Usage:   "wallet type, one of: HighloadV3, V4R2",
			},
			&cli.BoolFlag{
				Name:  ttac.ArgUseMEVProtection,
				Usage: "Use MEV protected submission",
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Err(err).Msg("client terminated with an error")
	} else {
		log.Info().Msg("client terminated without errors")
	}
}
