package ttac

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

// SendTransaction sends a signed transaction to the TON network via the specified endpoint.
//
// This function builds a request for submitting a signed TON transaction based on the wallet type.
// It converts the provided external message into a format that can be sent to the TON Trader API.
// The function supports different wallet types and encodes the transaction into a base64 format before
// sending it over HTTP.
//
// Parameters:
//   - ctx: A context.Context for managing the request's lifecycle, supporting cancellation and timeouts.
//   - endPoint: The API endpoint URL to which the transaction will be submitted (TON Trader API endpoint).
//   - authHeader: The authorization header required for authenticating the request.
//   - w: A pointer to the wallet.Wallet that holds the wallet used for this transaction.
//   - ext: A pointer to the tlb.ExternalMessage that contains the signed transaction to be submitted.
//
// Returns:
//   - string: The hash of the message body (msg_body_hash) if the transaction is successfully submitted.
//   - error: An error if the wallet type is unsupported, if the message fails to convert to a cell,
//     or if the submission to the API fails.
//
// Wallet Types Supported:
//   - HighloadV2R2
//   - HighloadV3
//   - V4R2
//   - V5R1Final
//
// Errors:
//   - Returns an error if the wallet type is unsupported (e.g., V3, V5R1Beta).
//   - Returns an error if the external message cannot be converted into a cell for submission.
//   - Returns an error if the HTTP submission to the TON Trader API fails.
//
// Example usage:
//
//	hash, err := SendTransaction(ctx, "https://example.com/submit", "<authHeader>", myWallet, extMsg)
//	if err != nil {
//	    log.Fatalf("Failed to send transaction: %v", err)
//	}
//	fmt.Printf("Transaction submitted successfully, message body hash: %s\n", hash)
func SendTransaction(ctx context.Context, endPoint, authHeader string, w *wallet.Wallet, ext *tlb.ExternalMessage) (string, error) {
	var walletType string
	switch w.GetSpec().(type) {
	case (*wallet.SpecHighloadV2R2):
		walletType = "HighloadV2R2"
	case (*wallet.SpecHighloadV3):
		walletType = "HighloadV3"
	case (*wallet.SpecV3):
		return "", errors.New("unsupported wallet type; please use one these: HighloadV2R2, HighloadV3, V4R2 or V5R1Final")
	case (*wallet.SpecV4R2):
		walletType = "V4R2"
	case (*wallet.SpecV5R1Beta):
		return "", errors.New("unsupported wallet type; please use one these: HighloadV2R2, HighloadV3, V4R2 or V5R1Final")
	case (*wallet.SpecV5R1Final):
		walletType = "V5R1Final"
	default:
		return "", errors.New("unsupported wallet type; please use one these: HighloadV2R2, HighloadV3, V4R2 or V5R1Final")
	}

	req := &TTASubmitRequest{
		Wallet: walletType,
	}
	extCell, err := tlb.ToCell(ext)
	if err != nil {
		return "", fmt.Errorf("failed to convert external message to cell: %w", err)
	}
	req.Transaction.Content = base64.StdEncoding.EncodeToString(extCell.ToBOC())
	res, err := submitTransaction(ctx, endPoint, authHeader, req, 10*time.Second)
	if err != nil {
		return "", err
	}
	return res.MsgBodyHash, err
}
