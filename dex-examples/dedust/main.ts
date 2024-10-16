import { Sender, internal, beginCell } from "@ton/core"
import { Address, toNano, TonClient4, TonClient4Parameters, WalletContractV3R1, WalletContractV3R2, WalletContractV4 } from "@ton/ton";
import { mnemonicToPrivateKey } from "@ton/crypto";
import { Asset, Factory, JettonRoot, MAINNET_FACTORY_ADDR, Pool, PoolType, VaultNative } from "@dedust/sdk";
import axios, { AxiosInstance } from "axios"

type BXParams = {
    endpoint: string;
    walletType: string;
    authKey: string;
    timeout?: number;
};

class BXTonClient extends TonClient4 {

    #bxEndpoint: string;
    #walletType: string;
    #authKey: string;
    #bxAxios: AxiosInstance;
    #timeout: number;

    constructor(tonArgs: TonClient4Parameters, bxArgs: BXParams) {
        super(tonArgs);
        this.#bxEndpoint = bxArgs.endpoint
        this.#authKey = bxArgs.authKey
        this.#walletType = bxArgs.walletType
        this.#bxAxios = axios.create();
        this.#timeout = bxArgs.timeout || 5000
    }

    // Overriding send method
    async sendMessage(message: Buffer) {
        let res = await this.#bxAxios.post(this.#bxEndpoint + '/api/v2/submit',
            { wallet: this.#walletType, transaction: { content: message.toString('base64') } },
            { headers: { 'Authorization': this.#authKey }, timeout: this.#timeout });
        if (res.status != 200) {
            throw Error(`Submission failure: ${res.data}`);
        }
        console.log(res.data)
        return { status: res.status };
    }
}

async function main() {
    if (!process.env.MNEMONIC) {
        throw new Error("Environment variable MNEMONIC is required.");
    }
    const mnemonic = process.env.MNEMONIC.split(" ");
    if (!process.env.AUTH_KEY) {
        throw new Error("Environment variable AUTH_KEY is required.");
    }
    const authKey = process.env.AUTH_KEY;
    if (!process.env.BX_TIP_ADDR) {
        throw new Error("Environment variable BX_TIP_ADDR is required.");
    }
    const bxTipAddr = process.env.BX_TIP_ADDR;

    const tonArgs = { endpoint: "https://mainnet-v4.tonhubapi.com", timeout: 10000 }
    const bxArgs = { endpoint: "https://frankfurt.ton.dex.blxrbdn.com", authKey: authKey, walletType: "V4R2" }
    const tonClient = new BXTonClient(tonArgs, bxArgs);

    // 1. Find a factory contract
    const factory = tonClient.open(
        Factory.createFromAddress(MAINNET_FACTORY_ADDR),
    );
    // 2. Opening your wallet
    const keys = await mnemonicToPrivateKey(mnemonic);
    const wallet = tonClient.open(
        WalletContractV4.create({ // Change to your wallet's type
            workchain: 0,
            publicKey: keys.publicKey,
        }),
    );
    // 3. Find a volatile pool TON/SCALE
    const scaleAddr = Address.parse(
        "EQBlqsm144Dq6SjbPI4jjZvA1hqTIP3CvHovbIfW_t-SCALE", // Scale jetton
    );
    const scale = tonClient.open(JettonRoot.createFromAddress(scaleAddr));
    const poolAddress = await factory.getPoolAddress({
        poolType: PoolType.VOLATILE,
        assets: [Asset.native(), Asset.jetton(scale.address)],
    })
    const pool = tonClient.open(
        Pool.createFromAddress(poolAddress),
    );
    console.log(pool.address)
    // 4. Find a vault for TON
    const nativeVault = tonClient.open(
        VaultNative.createFromAddress(
            await factory.getVaultAddress(Asset.native()),
        ),
    );
    // 5. Check if pool exists
    const lastBlock = await tonClient.getLastBlock();
    const poolState = await tonClient.getAccountLite(
        lastBlock.last.seqno,
        pool.address,
    );
    if (poolState.account.state.type !== "active") {
        throw new Error("Pool is not exist.");
    }
    // 6. Check if vault exists
    const vaultState = await tonClient.getAccountLite(
        lastBlock.last.seqno,
        nativeVault.address,
    );
    if (vaultState.account.state.type !== "active") {
        throw new Error("Native Vault is not exist.");
    }

    // 7. Estimate expected output amount
    const amountIn = toNano("0.003");
    const { amountOut: expectedAmountOut } = await pool.getEstimatedSwapOut({
        assetIn: Asset.native(),
        amountIn,
    });

    // Slippage handling (1%)
    const minAmountOut = (expectedAmountOut * 99n) / 100n; // expectedAmountOut - 1%
    console.log(minAmountOut)
    // 8. Send a transaction
    const bxSender: Sender = {
        send: async (args) => {
            let seqno = await wallet.getSeqno();
            let secretKey = keys.secretKey
            let transfer = wallet.createTransfer({
                seqno,
                secretKey,
                sendMode: args.sendMode,
                messages: [internal({
                    to: args.to,
                    value: args.value,
                    init: args.init,
                    body: args.body,
                    bounce: args.bounce
                }), internal({
                    to: bxTipAddr, // bloXroute tip
                    value: '0.005',
                    init: null,
                    body: beginCell().endCell(),
                    bounce: args.bounce
                })]
            });
            await wallet.send(transfer);
        }
    }
    await nativeVault.sendSwap(
        bxSender,
        {
            poolAddress: pool.address,
            amount: amountIn,
            limit: minAmountOut,
            gasAmount: toNano("0.25"),
        },
    );
}

main();