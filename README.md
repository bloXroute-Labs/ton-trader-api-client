# TON Trader API Client

**TON support and all related components will be sunset on Apr 24 2026**

## Introduction

This repository contains two example clients for interacting with BloXroute's TON Trader API service:

- **rawcli**: A low-level client that constructs and sends raw HTTP requests to the TON Trader API
- **oapicli**: A high-level client that uses a code-generated OpenAPI client to interact with all available API methods

Both clients demonstrate how to build and send TON transactions, including tip transfers, using different approaches.

---

## Building

If you have `go` and `make` installed, simply run:

```
make
```

This will build both example clients.

---

## Usage

### rawcli

The `rawcli` client (`ttc`) sends requests to the TON Trader API using raw HTTP calls.  
You will need a file containing the seed phrase for the wallet you wish to send from, and a valid bloXroute authorization header.

**Example command:**

```
go run cmd/rawcli/main.go \
  -c "your transfer comment" \
  -a 9800000 \
  -ah <your_auth_header> \
  -uri <ton_trader_api_endpoint> \
  -ll debug \
  --random-pause 23 \
  -rpc <ton_rpc_config_path> \
  -w1 <wallet1_seed_file> \
  -w2 <wallet2_seed_file> \
  -tip 8000008
```

**Key arguments:**

- `-a`, `--amount` — Amount to send (in nanotons)
- `-ah`, `--auth-header` — bloXroute auth header
- `-c`, `--comment` — Transfer comment
- `-uri` — TON Trader API endpoint
- `-ll`, `--log-level` — Log level (debug, info, warn, error)
- `--random-pause` — Random pause before sending (seconds)
- `-rpc` — TON RPC config file path
- `-w1` — Path to first wallet's seed phrase file
- `-w2` — Path to second wallet's seed phrase file (optional)
- `-tip` — Tip amount (in nanotons)

Run `go run cmd/rawcli/main.go -h` for the full list of options.

---

### oapicli

The `oapicli` client uses the OpenAPI-generated Go client for the TON Trader API, supporting all API methods.

**Example command:**

```
go run cmd/oapicli/main.go \
  --auth-header=<your_auth_header> \
  --wallet-path=<wallet_seed_file> \
  --destination-address=<destination_ton_address> \
  --amount=1000000 \
  --ton-rpc-uri=<ton_rpc_uri> \
  --trader-api=<ton_trader_api_endpoint> \
  --wallet-type=HighloadV3 \
  --tip=8000000 \
  --log-level=info
```

**Key arguments:**

- `--auth-header` — bloXroute auth header
- `--wallet-path` — Path to wallet seed phrase file
- `--destination-address` — TON address to send to
- `--amount` — Amount to send (in nanotons)
- `--ton-rpc-uri` — TON RPC URI or config file
- `--trader-api` — TON Trader API endpoint
- `--wallet-type` — Wallet type (HighloadV3, V4R2, etc)
- `--tip` — Tip amount (in nanotons)
- `--log-level` — Log level (debug, info, warn, error)
- `--mev-protection` — Enable MEV protection (optional)

Run `go run cmd/oapicli/main.go --help` for the full list of options.

---

Both clients support a variety of command line arguments for full control over transaction construction and API interaction.  
See the example commands above and use the `-h` or `--help` flag for each client to view all supported options.

---

## DEX Examples

See the `dex-examples` directory for code samples that connect the TON Trader API with popular DEXs such as DeDust and StonFi.

---