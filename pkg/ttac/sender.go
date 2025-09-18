package ttac

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

const (
	ArgAmount             = "amount"
	ArgAuthHeader         = "auth-header"
	ArgComment            = "comment"
	ArgDestinationAddress = "destination-address"
	ArgEndPointURI        = "uri"
	ArgLogLevel           = "log-level"
	ArgRandomAddon        = "random-addon"
	ArgRandomPause        = "random-pause"
	ArgTip                = "tip"
	ArgTonRPCURI          = "ton-rpc-uri"
	ArgWallet1            = "wallet-1"
	ArgWallet2            = "wallet-2"
	ArgWalletType         = "wallet-type"
	ArgUseMEVProtection   = "use-mev-protection"

	baseChainWorkchainNum = 0
)

var (
	defaultTimeout uint32 = 300
	seqNum         int64
)

type TxnsSummary struct {
	SentTime     time.Time
	MsgBodyHash  string
	RootCellHash string
}

type RunResult struct {
	TxnsSummary TxnsSummary
}

// RunWithConfig is a wrapper function that can be called from external packages
// to run the raw CLI functionality with the provided configuration.
func RunWithConfig(
	authHeader, endpointURI, tonRPCURI, wallet1Path, wallet2Path, destinationAddress, walletType, logLevel, comment string,
	amount, randomAddon, tip int64, randomPause uint, useMEVProtection bool,
) (*RunResult, error) {
	// Create a flag set with all the flags
	flagSet := &flag.FlagSet{}
	flagSet.String(ArgAuthHeader, authHeader, "")
	flagSet.String(ArgEndPointURI, endpointURI, "")
	flagSet.String(ArgTonRPCURI, tonRPCURI, "")
	flagSet.String(ArgWallet1, wallet1Path, "")
	flagSet.String(ArgWallet2, wallet2Path, "")
	flagSet.String(ArgDestinationAddress, destinationAddress, "")
	flagSet.String(ArgWalletType, walletType, "")
	flagSet.String(ArgLogLevel, logLevel, "")
	flagSet.String(ArgComment, comment, "")
	flagSet.Int64(ArgAmount, amount, "")
	flagSet.Int64(ArgRandomAddon, randomAddon, "")
	flagSet.Int64(ArgTip, tip, "")
	flagSet.Uint(ArgRandomPause, randomPause, "")
	flagSet.Bool(ArgUseMEVProtection, useMEVProtection, "")

	// Create a new app with empty flags since we'll set the values directly
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: ArgAuthHeader},
			&cli.StringFlag{Name: ArgEndPointURI},
			&cli.StringFlag{Name: ArgTonRPCURI},
			&cli.StringFlag{Name: ArgWallet1},
			&cli.StringFlag{Name: ArgWallet2},
			&cli.StringFlag{Name: ArgDestinationAddress},
			&cli.StringFlag{Name: ArgWalletType},
			&cli.StringFlag{Name: ArgLogLevel},
			&cli.StringFlag{Name: ArgComment},
			&cli.Int64Flag{Name: ArgAmount},
			&cli.Int64Flag{Name: ArgRandomAddon},
			&cli.Int64Flag{Name: ArgTip},
			&cli.UintFlag{Name: ArgRandomPause},
			&cli.BoolFlag{Name: ArgUseMEVProtection},
		},
	}

	// Create a context with the flag set
	cc := cli.NewContext(app, flagSet, nil)

	// Call the original run function
	return Run(cc)
}

func Run(cc *cli.Context) (*RunResult, error) {
	switch strings.ToLower(cc.String(ArgLogLevel)) {
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
	api, err := initTon(ctx, cc.String(ArgTonRPCURI))
	if err != nil {
		return nil, err
	}

	// get current master chain block
	info, err := api.GetMasterchainInfo(ctx)
	if err != nil || info == nil {
		return nil, fmt.Errorf("failed to obtain master chain info, %v", err)
	}

	if cc.String(ArgDestinationAddress) != "" && cc.String(ArgWallet2) != "" {
		return nil, fmt.Errorf("please use either -%s or -%s but not both", ArgDestinationAddress, ArgWallet2)
	}
	// initialize wallet from seed phrase
	ws, err := getWallets(api, [2]string{cc.String(ArgWallet1), cc.String(ArgWallet2)}, cc.String(ArgWalletType))
	if err != nil {
		return nil, err
	}

	for _, w := range ws {
		if w == nil {
			continue
		}

		// get and print wallet balance
		balance, err := w.GetBalance(ctx, info)
		if err != nil {
			return nil, fmt.Errorf("failed to obtain wallet balance, %v", err)
		}
		log.Info().Msgf("%v balance: %v", w.Address().String(), balance)
	}

	prg := rand.New(rand.NewSource(time.Now().UnixNano()))
	if cc.Uint(ArgRandomPause) > 0 {
		waitPeriod := time.Duration(prg.Intn(int(cc.Uint(ArgRandomPause))))
		log.Info().Msgf("pausing for %d seconds", waitPeriod)
		time.Sleep(waitPeriod * time.Second)
	}

	amount := cc.Int64(ArgAmount)
	if cc.Int64(ArgRandomAddon) > 0 {
		addOn := int64(prg.Intn(int(cc.Int64(ArgRandomAddon))))
		log.Info().Msgf("random addon: %v", addOn)
		amount += addOn
	}

	// generate the transaction: 1 transfer to destination address + a bloXroute tip transfer
	from, tx, err := genTx(ctx, api, ws, cc.String(ArgDestinationAddress), amount, cc.Int64(ArgTip), cc.String(ArgComment))
	if err != nil {
		return nil, err
	}

	// send transaction to TON trader API
	hash, sentTime, err := SendTransaction(ctx, cc.String(ArgEndPointURI), cc.String(ArgAuthHeader), from, tx, cc.Bool(ArgUseMEVProtection))
	if err != nil {
		return nil, err
	}

	c, err := tlb.ToCell(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize external message: %w", err)
	}

	// Compute the cell's representation hash
	h := hex.EncodeToString(c.Hash())

	log.Info().Msgf("tx sent, msg body hash: %s, root cell hash: %v", hash, h)
	return &RunResult{
		TxnsSummary{
			SentTime:     sentTime,
			MsgBodyHash:  hash,
			RootCellHash: h,
		},
	}, nil
}

func logArgs(cc *cli.Context) {
	args := []string{
		ArgAmount,
		ArgAuthHeader,
		ArgComment,
		ArgDestinationAddress,
		ArgEndPointURI,
		ArgLogLevel,
		ArgRandomAddon,
		ArgRandomPause,
		ArgTip,
		ArgTonRPCURI,
		ArgWallet1,
		ArgWallet2,
		ArgWalletType,
		ArgUseMEVProtection,
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

func getWallets(api *ton.APIClient, paths [2]string, walletType string) ([2]*wallet.Wallet, error) {
	var (
		err error
		res [2]*wallet.Wallet
	)
	for i, path := range paths {
		if path == "" {
			continue
		}
		res[i], err = getWallet(api, path, walletType)
		if err != nil {
			return res, err
		}
	}
	return res, nil
}

func getWallet(api *ton.APIClient, path, walletType string) (*wallet.Wallet, error) {
	phrase, err := readPhrase(path)
	if err != nil {
		return nil, err
	}
	wallets := map[string]wallet.Version{
		"HighloadV3": wallet.HighloadV3,
		"V4R2":       wallet.V4R2,
		"V5R1Final":  wallet.V5R1Final,
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
			Workchain:       baseChainWorkchainNum,
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
		tx, err = GenerateTransaction(ctx, from, to.Address().String(), amount, tip, nil, comment)
		if err != nil {
			return nil, nil, err
		}
		return from, tx, err
	}
	if !firstWallet {
		return nil, nil, fmt.Errorf("first wallet is nil")
	}
	// we specify just one wallet and want to send from to the destination address
	tx, err = GenerateTransaction(ctx, ws[0], toAddress, amount, tip, nil, comment)
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
