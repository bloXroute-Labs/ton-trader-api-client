# TON trader API client

## intro

This repo contains example code that demonstrates how to build and send a request to bloXroute's TON trader API service.
In order to try the code you will need: a
- file that contains the seed phrase for the wallet from which you wish to send
- bloXroute authorization header

## how to build

If you have `go` and `make` installed on your system simply call the `make` command.


## how to invoke

example:

```
bin/ttc --auth-header <authHeader> --from-wallet ./ton-1 --destination-address UQAyHtxcmMSOUuqqPF1QUbUSTIAJz5bOPKBP-5a4ZtgEQxRt -a 220000000
```

## command line arguments

Last but not least call the client with `-h` (`bin/ttc -h`) to view all supported command line arguments

```
NAME:
   TON trader API client - make requests to ton-trader-api service

USAGE:
   TON trader API client [global options] command [command options]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --amount value, -a value                  amount, default: 0.25 TON (default: 250000000)
   --auth-header value, --ah value           bloXroute auth header
   --comment value, -c value                 transfer comment (default: "TON trader API test, 2024-09-19T19:36:45.864285")
   --destination-address value, --tda value  transaction destination address
   --uri value                               TON trader API endpoint (default: "https://eu.ton.dex.blxrbdn.com")
   --from-wallet value, --fw value           file with the seed phrase for the sending wallet
   --log-level value, --ll value             log level, one of: debug, info, warn, error (default: "info")
   --tip value, -t value                     tip, default: 0.015 TON (default: 15000000)
   --ton-rpc-uri value, --rpc value          file with the seed phrase for the receiving wallet (default: "https://ton.org/global-config.json")
   --help, -h                                show help
```
