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

// GetTipTransfer creates a transfer message for a tip transaction.
//
// This function generates a tip transaction message from the given wallet,
// transferring the specified amount of TONs to a hardcoded tip address.
//
// Parameters:
//   - from: A pointer to the wallet.Wallet from which the tip will be sent.
//   - tip: The tip amount in nanoTON (int64).
//
// Returns:
//   - *wallet.Message: A pointer to the generated transfer message.
//   - error: An error if the tip address parsing or transfer message creation fails.
//
// Errors:
//   - Returns an error if the tip address cannot be parsed or the transfer message
//     cannot be generated.
//
// Example usage:
//
//	msg, err := GetTipTransfer(myWallet, 15000000)
//	if err != nil {
//	    log.Fatalf("Failed to create tip transfer: %v", err)
//	}
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

// GenerateTransaction creates a new transaction with the specified amount and tip, including an optional comment.
//
// This function generates a transaction from the given wallet to the specified destination address.
// It also includes a tip transfer message and allows adding an optional comment. The transaction
// is built as an external message, ready to be submitted to the blockchain.
//
// Parameters:
//   - ctx: A context.Context for controlling the request's lifecycle.
//   - from: A pointer to the wallet.Wallet from which the transaction will be sent.
//   - to: The destination address as a string (TON address format).
//   - amount: The amount of TONs to be transferred, in nanoTON (int64).
//   - tip: The tip amount to be included in the transaction, in nanoTON (int64).
//   - comment: A string comment to be attached to the transaction.
//
// Returns:
//   - *tlb.ExternalMessage: A pointer to the built external message that can be submitted to the network.
//   - error: An error if any part of the transaction (parsing address, building transfer, etc.) fails.
//
// Errors:
//   - Returns an error if the destination address cannot be parsed.
//   - Returns an error if building the transfer message for the transaction or the tip transfer fails.
//   - Returns an error if the external message cannot be built.
//
// Example usage:
//
//	extMsg, err := GenerateTransaction(ctx, myWallet, "EQD...", 250000000, 15000000, "Test transaction")
//	if err != nil {
//	    log.Fatalf("Failed to generate transaction: %v", err)
//	}
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
