package ttac

import (
	"context"
	"fmt"
	"math/big"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

const tipAddress = "UQAw0AJjHbMYQobYXHBoW29ShKx1V2UjaiKanhDYBNJYDPUh"

// GetTipTransfer generates a transfer to the bloXroute tip address
func GetTipTransfer(from *wallet.Wallet, tip int64) (*wallet.Message, error) {
	tipAmount := tlb.FromNanoTON(big.NewInt(tip))
	tipAddress, err := address.ParseAddr(tipAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tip address: %v", err)
	}
	tt, err := from.BuildTransfer(tipAddress, tipAmount, true, fmt.Sprintf("tip from %s", from.Address().String()))
	if err != nil {
		return nil, fmt.Errorf("failed to generate tip transfer: %v", err)
	}
	return tt, nil
}

// GenerateTransaction generates a TON transaction including a tip transfer to the bloXroute tip address
func GenerateTransaction(ctx context.Context, from *wallet.Wallet, to string, amount, tip int64, comment string) (*tlb.ExternalMessage, error) {
	var msgs []*wallet.Message

	toAddress, err := address.ParseAddr(to)
	if err != nil {
		return nil, fmt.Errorf("failed to parse destination address: %v", err)
	}
	t1, err := from.BuildTransfer(toAddress, tlb.FromNanoTON(big.NewInt(amount)), true, comment)
	if err != nil {
		return nil, fmt.Errorf("failed to generate transaction: %v", err)
	}
	msgs = append(msgs, t1)

	tt, err := GetTipTransfer(from, tip)
	if err != nil {
		return nil, err
	}
	msgs = append(msgs, tt)
	ext, err := from.BuildExternalMessageForMany(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("failed to build external message: %v", err)
	}
	return ext, nil
}
