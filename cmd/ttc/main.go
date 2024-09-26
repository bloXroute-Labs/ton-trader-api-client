package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"math/rand"

	"github.com/bloXroute-Labs/ton-trader-api-client/pkg/ttac"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

const (
	argAmount             = "amount"
	argAuthHeader         = "auth-header"
	argComment            = "comment"
	argDestinationAddress = "destination-address"
	argEndPointURI        = "uri"
	argFromWallet         = "from-wallet"
	argLogLevel           = "log-level"
	argRandomAddon        = "random-addon"
	argRandomPause        = "random-pause"
	argTip                = "tip"
	argTonRPCURI          = "ton-rpc-uri"
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
		Name:   "TON trader API client",
		Usage:  "make requests to ton-trader-api service",
		Action: run,
		Flags: []cli.Flag{
			&cli.Int64Flag{
				Name:    argAmount,
				Aliases: []string{"a"},
				Value:   250000000,
				Usage:   "amount, default: 0.25 TON",
			},
			&cli.StringFlag{
				Name:     argAuthHeader,
				Aliases:  []string{"ah"},
				Required: true,
				Usage:    "bloXroute auth header",
			},
			&cli.StringFlag{
				Name:    argComment,
				Aliases: []string{"c"},
				Value:   fmt.Sprintf("TON trader API test, %s", time.Now().UTC().Format(zerolog.TimeFieldFormat)),
				Usage:   "transfer comment",
			},
			&cli.StringFlag{
				Name:     argDestinationAddress,
				Aliases:  []string{"tda"},
				Required: true,
				Usage:    "transaction destination address",
			},
			&cli.StringFlag{
				Name:  argEndPointURI,
				Value: "https://frankfurt.ton.dex.blxrbdn.com",
				Usage: "TON trader API endpoint",
			},
			&cli.StringFlag{
				Name:     argFromWallet,
				Aliases:  []string{"fw"},
				Required: true,
				Usage:    "file `path` with the seed phrase for the sending wallet",
			},
			&cli.StringFlag{
				Name:    argLogLevel,
				Aliases: []string{"ll"},
				Value:   "info",
				Usage:   "log `level`, one of: debug, info, warn, error",
			},
			&cli.Int64Flag{
				Name:    argRandomAddon,
				Aliases: []string{"ra"},
				Value:   2500000,
				Usage:   "random `addon` to the specified amount, default: 0.0025 TON",
			},
			&cli.UintFlag{
				Name:    argRandomPause,
				Aliases: []string{"rp"},
				Usage:   "random `pause` to take before sending, in seconds",
			},
			&cli.Int64Flag{
				Name:    argTip,
				Aliases: []string{"t"},
				Value:   15000000,
				Usage:   "`tip`, default: 0.015 TON",
			},
			&cli.StringFlag{
				Name:    argTonRPCURI,
				Aliases: []string{"rpc"},
				Value:   "https://ton.org/global-config.json",
				Usage:   "TON RPC configuration to use",
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Err(err).Msg("client terminated with an error")
	} else {
		log.Info().Msg("client terminated without errors")
	}
}

func run(cc *cli.Context) error {
	switch strings.ToLower(cc.String(argLogLevel)) {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	}
	logArgs(cc)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// initialize TON API
	api, err := initTon(ctx, cc.String(argTonRPCURI))
	if err != nil {
		return err
	}

	// initialize wallet from seed phrase
	from, err := getWallet(api, cc.String(argFromWallet))
	if err != nil {
		return err
	}

	// get current master chain block
	info, err := api.GetMasterchainInfo(ctx)
	if err != nil || info == nil {
		return fmt.Errorf("failed to obtain master chain info, %v", err)
	}
	// get and print wallet balance
	balance, err := from.GetBalance(ctx, info)
	if err != nil {
		return fmt.Errorf("failed to obtain wallet balance, %v", err)
	}
	log.Info().Msgf("wallet balance: %v", balance)

	prg := rand.New(rand.NewSource(time.Now().UnixNano()))
	if cc.Uint(argRandomPause) > 0 {
		waitPeriod := time.Duration(prg.Intn(int(cc.Uint(argRandomPause))))
		log.Info().Msgf("pausing for %d seconds", waitPeriod)
		time.Sleep(waitPeriod * time.Second)
	}

	amount := cc.Int64(argAmount)
	if cc.Int64(argRandomAddon) > 0 {
		addOn := int64(prg.Intn(int(cc.Int64(argRandomAddon))))
		log.Info().Msgf("random addon: %v", addOn)
		amount += addOn
	}

	// generate the transaction: 1 transfer to destination address + a bloXroute tip transfer
	tx, err := ttac.GenerateTransaction(ctx, from, cc.String(argDestinationAddress), amount, cc.Int64(argTip), cc.String(argComment))
	if err != nil {
		return err
	}

	// send transaction to TON trader API
	hash, err := ttac.SendTransaction(ctx, cc.String(argEndPointURI), cc.String(argAuthHeader), from, tx)
	if err != nil {
		return err
	}

	log.Info().Msgf("tx sent, msg body hash: %s", hash)
	return nil
}

func logArgs(cc *cli.Context) {
	args := []string{
		argAmount,
		argAuthHeader,
		argComment,
		argDestinationAddress,
		argEndPointURI,
		argFromWallet,
		argLogLevel,
		argRandomAddon,
		argRandomPause,
		argTip,
		argTonRPCURI,
	}
	for _, arg := range args {
		log.Info().Msgf("%s = %v", arg, cc.Value(arg))
	}
}

func initTon(ctx context.Context, rpcURI string) (*ton.APIClient, error) {
	var (
		cfg *liteclient.GlobalConfig
		err error
	)
	client := liteclient.NewConnectionPool()
	if strings.HasPrefix(rpcURI, "http") {
		cfg, err = liteclient.GetConfigFromUrl(ctx, rpcURI)
	} else {
		cfg, err = liteclient.GetConfigFromFile(rpcURI)
	}
	if err != nil {
		return nil, err
	}
	err = client.AddConnectionsFromConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	api := ton.NewAPIClient(client)
	api.SetTrustedBlockFromConfig(cfg)
	return api, nil
}

func getWallet(api *ton.APIClient, path string) (*wallet.Wallet, error) {
	phrase, err := readPhrase(path)
	if err != nil {
		return nil, err
	}
	return wallet.FromSeed(api, phrase, wallet.V4R2)
}

func readPhrase(path string) ([]string, error) {
	phrase, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read phrase from '%s', %v", path, err)
	}
	words := strings.Fields(string(phrase))
	if len(words) != 24 {
		return nil, fmt.Errorf("invalid phrase, length %d", len(words))
	}
	return words, nil
}
