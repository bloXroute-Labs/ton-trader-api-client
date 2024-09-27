# TON trader API DeDust example
The given example creates a DeDust swap on TON/SCALE pool and sends it using TON Trader API endpoint to ensure fast and reliable landing. See also https://docs.dedust.io/recipes

## Installation and run command
```bash
npm install
npm run build; MNEMONIC="mnemonic phrase of your wallet" AUTH_KEY="your bloXroute API key" BX_TIP_ADDR="bloXroute tip address" npm run start
```

Output example:
```bash
> node dist/main.js

EQDcm06RlreuMurm-yik9WbL6kI617B77OrSRF_ZjoCYFuny
4653733n
{
  msg_body_hash: '0x855c12ff1f373bd1c6cxd45114eb559b83ae6ff89366041c7a022587ecd724a0'
}
```
