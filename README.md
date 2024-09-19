# TON trader API client

## intro

This repo contains example code that demonstrates how to build and send a request to bloXroute's TON trader API service.
In order to try the code you will need: a
- file that contains the seed phrase for the wallet from which you wish to send
- bloXroute authorization header

## how to build

if you have `go` and `make` installed on your system simply call the `make` command.


## how to invoke

example:

```
bin/ttc --auth-header <authHeader> --from-wallet ./ton-1 --destination-address UQAyHtxcmMSOUuqqPF1QUbUSTIAJz5bOPKBP-5a4ZtgEQxRt -a 220000000
```
