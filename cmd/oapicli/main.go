package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bloXroute-Labs/ton-trader-api-client/api"
	"github.com/bloXroute-Labs/ton-trader-api-client/pkg/ttac"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

const (
	argTonRPCURI     = "ton-rpc-uri"
	argTraderAPI     = "trader-api"
	argAuthHeader    = "auth-header"
	argLogLevel      = "log-level"
	argTipAmount     = "tip"
	argWalletPath    = "wallet-path"
	argWalletType    = "wallet-type"
	argAmount        = "amount"
	argMevProtection = "mev-protection"
	argDestAddress   = "destination-address"
)

func main() {
	zerolog.TimestampFieldName = "t"
	zerolog.LevelFieldName = "l"
	zerolog.MessageFieldName = "m"
	zerolog.TimeFieldFormat = "2006-01-02T15:04:05.000000"

	app := &cli.App{
		Name:  "ton-trader-oapi-cli",
		Usage: "Send TON transactions via ton-trader-api",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: argTonRPCURI, Required: true, Usage: "TON RPC URI"},
			&cli.StringFlag{Name: argTraderAPI, Required: true, Usage: "Ton Trader API endpoint"},
			&cli.StringFlag{Name: argAuthHeader, Required: true, Usage: "Ton Trader API auth header"},
			&cli.StringFlag{Name: argLogLevel, Value: "info", Usage: "Log level"},
			&cli.Int64Flag{Name: argTipAmount, Required: true, Usage: "Tip amount in nanotons"},
			&cli.StringFlag{Name: argWalletPath, Required: true, Usage: "Path to wallet seed file"},
			&cli.StringFlag{Name: argWalletType, Required: true, Usage: "Wallet type (HighloadV3, V4R2, etc)"},
			&cli.Int64Flag{Name: argAmount, Required: true, Usage: "Amount in nanotons to send"},
			&cli.BoolFlag{Name: argMevProtection, Usage: "Enable MEV protection"},
			&cli.StringFlag{Name: argDestAddress, Required: true, Usage: "Destination address"},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		log.Err(err).Msg("terminated with error")
	} else {
		log.Info().Msg("terminated without errors")
	}
}

func run(c *cli.Context) error {
	setLogLevel(c.String(argLogLevel))
	logArgs(c)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	tonClient, err := createTONClient(ctx, c.String(argTonRPCURI))
	if err != nil {
		return fmt.Errorf("failed to connect to TON RPC: %w", err)
	}
	senderWallet, err := readWallet(tonClient, c.String(argWalletType), c.String(argWalletPath))
	if err != nil {
		return fmt.Errorf("failed to read wallet of type %s from path %s: %w", c.String(argWalletType), c.String(argWalletPath), err)
	}
	ttaClient, err := api.NewClientWithResponses(c.String(argTraderAPI))
	if err != nil {
		return fmt.Errorf("failed to create ton-trader-api client: %w", err)
	}

	mevProtection := c.Bool(argMevProtection)
	tipAddr, err := getTipAddress(ctx, ttaClient, senderWallet, c.String(argAuthHeader), mevProtection)
	if err != nil {
		return fmt.Errorf("failed to get tip address: %w", err)
	}

	signedMessageB64, err := buildAndSignTransaction(
		ctx,
		c.String(argDestAddress),
		senderWallet,
		c.Int64(argAmount),
		tipAddr,
		c.Int64(argTipAmount),
	)
	if err != nil {
		return fmt.Errorf("failed to build/sign transaction: %w", err)
	}

	req := api.SubmitRequest{
		Wallet:           api.SubmitRequestWallet(c.String(argWalletType)),
		UseMevProtection: &mevProtection,
	}
	req.Transaction.Content = signedMessageB64
	params := &api.PostApiV2SubmitParams{Authorization: c.String(argAuthHeader)}
	resp, err := ttaClient.PostApiV2SubmitWithResponse(ctx, params, req)
	if err != nil {
		return fmt.Errorf("API call failed: %w", err)
	}
	if resp.JSON200 != nil {
		log.Info().Msgf("Transaction sent, msg body hash: %s", resp.JSON200.MsgBodyHash)
		return nil
	}
	if resp.JSON400 != nil {
		return fmt.Errorf("API error: %s", resp.JSON400.Message)
	}

	return fmt.Errorf("unexpected API response: %s", string(resp.Body))
}

func setLogLevel(lvl string) {
	switch strings.ToLower(lvl) {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	}
}

func logArgs(c *cli.Context) {
	for _, name := range c.FlagNames() {
		log.Info().Msgf("%s = %v", name, c.Value(name))
	}
}

func readWallet(tonClient ton.APIClientWrapped, walletType, walletPath string) (*wallet.Wallet, error) {
	seed, err := os.ReadFile(walletPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read wallet seed: %w", err)
	}
	seedWords := strings.Fields(string(seed))
	var w *wallet.Wallet
	switch strings.ToLower(walletType) {
	case "highloadv3":
		w, err = wallet.FromSeed(tonClient, seedWords, wallet.HighloadV3)
	case "v4r2":
		w, err = wallet.FromSeed(tonClient, seedWords, wallet.V4R2)
	case "v5r1final":
		w, err = wallet.FromSeed(tonClient, seedWords, wallet.V5R1Final)
	default:
		return nil, fmt.Errorf("unsupported wallet type: %s", walletType)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load wallet: %w", err)
	}
	return w, nil
}

func getTipAddress(ctx context.Context, ttaClient api.ClientWithResponsesInterface, senderWallet *wallet.Wallet, authHeader string, mevProtection bool) (*address.Address, error) {
	var tipAddrStr string
	if mevProtection {
		wa := senderWallet.Address().String()
		tipReq := api.TipWalletRequest{Address: &wa} // Note that you can pass either shard or address, or base64-encoded transaction
		tipParams := &api.PostApiV2TipWalletParams{Authorization: authHeader}
		tipResp, err := ttaClient.PostApiV2TipWalletWithResponse(ctx, tipParams, tipReq)
		if err != nil {
			return nil, fmt.Errorf("PostApiV2TipWalletWithResponse failed: %w", err)
		}
		if tipResp.JSON200 == nil {
			return nil, fmt.Errorf("unexpected response from PostApiV2TipWallet: %s", string(tipResp.Body))
		}
		log.Info().Msgf("Shard: %s", tipResp.JSON200.Shard)
		tipAddrStr = tipResp.JSON200.TipWalletAddress
		log.Info().Msgf("Tip wallet address: %s", tipAddrStr)
	} else {
		tipParams := &api.GetApiV2TipWalletParams{Authorization: authHeader}
		tipResp, err := ttaClient.GetApiV2TipWalletWithResponse(ctx, tipParams)
		if err != nil {
			return nil, fmt.Errorf("GetApiV2TipWalletWithResponse failed: %w", err)
		}
		if tipResp.JSON200 == nil {
			return nil, fmt.Errorf("unexpected response from GetApiV2TipWallet: %s", string(tipResp.Body))
		}
		tipAddrStr = tipResp.JSON200.TipWalletAddress
		log.Info().Msgf("Tip wallet address: %s", tipAddrStr)
	}
	tipAddr, err := address.ParseAddr(tipAddrStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tip wallet address: %w", err)
	}
	return tipAddr, nil
}

func buildAndSignTransaction(
	ctx context.Context,
	destAddr string,
	w *wallet.Wallet,
	amount int64,
	tipAddr *address.Address,
	tipAmount int64,
) (string, error) {
	extMsg, err := ttac.GenerateTransaction(ctx, w, destAddr, amount, tipAmount, tipAddr, "")
	if err != nil {
		return "", fmt.Errorf("failed to generate transaction: %w", err)
	}
	extCell, err := tlb.ToCell(extMsg)
	if err != nil {
		return "", fmt.Errorf("failed to convert external message to cell: %w", err)
	}

	return base64.StdEncoding.EncodeToString(extCell.ToBOC()), nil
}

func createTONClient(ctx context.Context, rpcURI string) (*ton.APIClient, error) {
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
