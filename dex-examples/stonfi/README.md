# TON trader API StonFi example
The given example creates a StonFi swap on TON/STON pool so that it can be send via TON Trader API endpoint to ensure fast and reliable landing. See also https://docs.ston.fi/docs/developer-section/sdk/dex-v1/swap

Note that this code is also using https://toncenter.com/api/v2/jsonRPC which is subject to rate limiting.

## Installation and run command
```bash
npm install
npm run build; MNEMONIC="mnemonic phrase of your wallet" WALLET_ADDR=="your wallet address" BX_TIP_ADDR="bloXroute tip address" npm run start
```

{to: Address, value: 191000000n, body: Cell}
main.ts:43
te6cckECBAEAAWUAAuOIAWHOZ6pdd/4M30ff288x/SVnUlXeCWmAc9orQ2582lOCBR0f8JBJf9CCfPfjedA34xKI7NSL5wml8DutN2Wx5xGdGRb4DeMAQK2BHTIxNQ/BTY0GVp3WdtreMbI1H5oI0HFNTRi7N7aBWAAAATgACAwBAwHTYgAIqFqMWTE1aoxM/MRD/EEluAMqKyKvv/FAn4CTTNIDD6BbE24AAAAAAAAAAAAAAAAAAA+KfqUAAAAAAAAwOTW42AgA7zuZAqJxsqAciTilI8/iTnGEeq62piAAHtRKd6wOcJwQLBuBAwIAlSWThWGAEPckSDVNSxvmJOKMGCE4qb+zE0NdQGXsIPpYy63RDznGW42BACw5zPVLrv/Bm+j7+3nmP6Ss6kq5wS0wDntFaG3Pm0pwUABmYgAYaAExjtmMIUNsLjg0LbepQlY6q7KRtRFNTwhsAmksBhh6EgAAAAAAAAAAAAAAAAAABzGaRA==

-- Now sending it using TON Trader API --
http POST https://frankfurt.ton.dex.blxrbdn.com/api/v2/submit wallet=V4R2 transaction\[content\]="te6cckECBAEAAWUAAuOIAWHOZ6pdd/4M30ff288x/SVnUlXeCWmAc9orQ2582lOCBR0f8JBJf9CCfPfjedA34xKI7NSL5wml8DutN2Wx5xGdGRb4DeMAQK2BHTIxNQ/BTY0GVp3WdtreMbI1H5oI0HFNTRi7N7aBWAAAATgACAwBAwHTYgAIqFqMWTE1aoxM/MRD/EEluAMqKyKvv/FAn4CTTNIDD6BbE24AAAAAAAAAAAAAAAAAAA+KfqUAAAAAAAAwOTW42AgA7zuZAqJxsqAciTilI8/iTnGEeq62piAAHtRKd6wOcJwQLBuBAwIAlSWThWGAEPckSDVNSxvmJOKMGCE4qb+zE0NdQGXsIPpYy63RDznGW42BACw5zPVLrv/Bm+j7+3nmP6Ss6kq5wS0wDntFaG3Pm0pwUABmYgAYaAExjtmMIUNsLjg0LbepQlY6q7KRtRFNTwhsAmksBhh6EgAAAAAAAAAAAAAAAAAABzGaRA==" Authorization: "your bloXroute API key"

HTTP/1.1 200 OK
Content-Length: 87
Content-Type: application/json
Date: Fri, 27 Sep 2024 15:32:23 GMT
X-Content-Type-Options: nosniff
X-Frame-Options: SAMEORIGIN
X-Xss-Protection: 1; mode=block

{
    "msg_body_hash": "message body hash"
}
```

## Using Golang
See the examples [in tongo](https://github.com/tonkeeper/tongo/blob/master/contract/stonfi/stonfi_swap.go).