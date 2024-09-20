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
