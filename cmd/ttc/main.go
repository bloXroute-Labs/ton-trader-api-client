package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"math/rand"

	"github.com/bloXroute-Labs/ton-trader-api-client/pkg/ttac"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

const (
	argAmount             = "amount"
	argAuthHeader         = "auth-header"
	argComment            = "comment"
	argDestinationAddress = "destination-address"
	argEndPointURI        = "uri"
	argLogLevel           = "log-level"
	argRandomAddon        = "random-addon"
	argRandomPause        = "random-pause"
	argTip                = "tip"
	argTonRPCURI          = "ton-rpc-uri"
	argWallet1            = "wallet-1"
	argWallet2            = "wallet-2"
	argWalletType         = "wallet-type"
)

var (
	bts, rev, version string
	defaultTimeout    uint32 = 300
	seqNum            int64
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
				Name:    argDestinationAddress,
				Aliases: []string{"tda"},
				Usage:   "transaction destination address",
			},
			&cli.StringFlag{
				Name:  argEndPointURI,
				Value: "https://frankfurt.ton.dex.blxrbdn.com",
				Usage: "TON trader API endpoint",
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
			&cli.StringFlag{
				Name:     argWallet1,
				Aliases:  []string{"w1"},
				Required: true,
				Usage:    "file `path` with the seed phrase for (sending?) wallet",
			},
			&cli.StringFlag{
				Name:    argWallet2,
				Aliases: []string{"w2"},
				Usage:   "file `path` with the seed phrase for second wallet",
			},
			&cli.StringFlag{
				Name:    argWalletType,
				Aliases: []string{"wt"},
				Value:   "V4R2",
				Usage:   "wallet type, one of: HighloadV3, V4R2",
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

	// get current master chain block
	info, err := api.GetMasterchainInfo(ctx)
	if err != nil || info == nil {
		return fmt.Errorf("failed to obtain master chain info, %v", err)
	}

	if cc.String(argDestinationAddress) != "" && cc.String(argWallet2) != "" {
		return fmt.Errorf("please use either -%s or -%s but not both", argDestinationAddress, argWallet2)
	}
	// initialize wallet from seed phrase
	ws, err := getWallets(api, info, [2]string{cc.String(argWallet1), cc.String(argWallet2)}, cc.String(argWalletType))
	if err != nil {
		return err
	}

	for _, w := range ws {
		// get and print wallet balance
		balance, err := w.GetBalance(ctx, info)
		if err != nil {
			return fmt.Errorf("failed to obtain wallet balance, %v", err)
		}
		log.Info().Msgf("%v balance: %v", w.Address().String(), balance)
	}

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
	from, tx, err := genTx(ctx, api, ws, cc.String(argDestinationAddress), amount, cc.Int64(argTip), cc.String(argComment))
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
		argLogLevel,
		argRandomAddon,
		argRandomPause,
		argTip,
		argTonRPCURI,
		argWallet1,
		argWallet2,
		argWalletType,
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

func mbf(_ context.Context, _ uint32) (uint32, int64, error) {
	requestId := uint32(atomic.AddInt64(&seqNum, 1))
	tm := time.Now().Unix() - 30
	return requestId, tm, nil
}

func getWallets(api *ton.APIClient, cmi *ton.BlockIDExt, paths [2]string, walletType string) ([2]*wallet.Wallet, error) {
	var (
		err error
		res [2]*wallet.Wallet
	)
	for i, path := range paths {
		if path == "" {
			continue
		}
		res[i], err = getWallet(api, cmi, path, walletType)
		if err != nil {
			return res, err
		}
	}
	return res, nil
}

func getWallet(api *ton.APIClient, cmi *ton.BlockIDExt, path, walletType string) (*wallet.Wallet, error) {
	phrase, err := readPhrase(path)
	if err != nil {
		return nil, err
	}
	wallets := map[string]wallet.Version{
		"HighloadV3": wallet.HighloadV3,
		"V4R2":       wallet.V4R2,
	}
	wt, ok := wallets[walletType]
	if !ok {
		return nil, fmt.Errorf("invalid wallet type: '%v'", walletType)
	}
	var w *wallet.Wallet
	switch wt {
	case wallet.HighloadV3:
		w, err = wallet.FromSeed(api, phrase, wallet.ConfigHighloadV3{
			MessageTTL:     defaultTimeout,
			MessageBuilder: mbf,
		})
	case wallet.V5R1Final:
		w, err = wallet.FromSeed(api, phrase, wallet.ConfigV5R1Final{
			NetworkGlobalID: wallet.MainnetGlobalID,
			Workchain:       int8(cmi.Workchain),
		})
	default:
		w, err = wallet.FromSeed(api, phrase, wt)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to instantiate %v wallet, %v", walletType, err)
	}
	return w, nil
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

func genTx(ctx context.Context, api *ton.APIClient, ws [2]*wallet.Wallet, toAddress string, amount, tip int64, comment string) (*wallet.Wallet, *tlb.ExternalMessage, error) {
	var (
		bothWallets bool = ws[0] != nil && ws[1] != nil
		err         error
		firstWallet bool = ws[0] != nil
		from, to    *wallet.Wallet
		tx          *tlb.ExternalMessage
	)

	// we specify 2 wallets and want to send from the wallet with the higher balance
	if bothWallets {
		from, to, err = determineSender(ctx, api, ws)
		if err != nil {
			return nil, nil, err
		}
		tx, err = ttac.GenerateTransaction(ctx, from, to.Address().String(), amount, tip, comment)
		if err != nil {
			return nil, nil, err
		}
		return from, tx, err
	}
	if !firstWallet {
		return nil, nil, fmt.Errorf("first wallet is nil")
	}
	// we specify just one wallet and want to send from to the destination address
	tx, err = ttac.GenerateTransaction(ctx, ws[0], toAddress, amount, tip, comment)
	if err != nil {
		return nil, nil, err
	}
	return ws[0], tx, err
}

func determineSender(ctx context.Context, api *ton.APIClient, ws [2]*wallet.Wallet) (*wallet.Wallet, *wallet.Wallet, error) {
	// the wallet with higher balance shall be the sender
	info, err := api.GetMasterchainInfo(ctx)
	if err != nil || info == nil {
		return nil, nil, fmt.Errorf("failed to obtain master chain info, %v", err)
	}

	b1, err := ws[0].GetBalance(ctx, info)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to obtain wallet balance for %v, %v", ws[0].Address(), err)
	}

	b2, err := ws[1].GetBalance(ctx, info)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to obtain wallet balance for %v, %v", ws[1].Address(), err)
	}

	if b1.Nano().Int64() > b2.Nano().Int64() {
		return ws[0], ws[1], nil
	}
	return ws[1], ws[0], nil
}
