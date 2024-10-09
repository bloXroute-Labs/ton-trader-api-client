import { Sender, internal, beginCell } from "@ton/core"
import { Address, toNano, TonClient, TonClient4, TonClient4Parameters, WalletContractV3R1, WalletContractV3R2, WalletContractV4 } from "@ton/ton";
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
        console.log(message.toString('base64'))
        // let res = await this.#bxAxios.post(this.#bxEndpoint + '/api/v2/submit',
        //     { wallet: this.#walletType, transaction: { content: message.toString('base64') } },
        //     { headers: { 'Authorization': this.#authKey }, timeout: this.#timeout });
        // if (res.status != 200) {
        //     throw Error(`Submission failure: ${res.data}`);
        // }
        // console.log(res.data)
        // return { status: res.status };
        return { status: 200 };
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
    if (!process.env.CLIENT) {
        throw new Error("Environment variable CLIENT is required.");
    }
    const client = process.env.CLIENT;    

    const tonArgs = { endpoint: "https://mainnet-v4.tonhubapi.com", timeout: 10000 }
    // const bxArgs = { endpoint: "https://frankfurt.ton.dex.blxrbdn.com", authKey: authKey, walletType: "HighloadV3" }
    const bxArgs = { endpoint: "http://localhost:8080", authKey: authKey, walletType: "HighloadV3" }
    const ton4Client = new BXTonClient(tonArgs, bxArgs);

    const tonClient = new TonClient({endpoint: "https://toncenter.com/api/v2/jsonRPC", apiKey: "f08f19ad07c90718ac69eb688290f248c6a46d5e74e5bfa790cea41a31476e54"});

    // 1. Find a factory contract
    const factory = tonClient.open(
        Factory.createFromAddress(MAINNET_FACTORY_ADDR),
    );
    const factory4 = ton4Client.open(
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
    const wallet4 = ton4Client.open(
        WalletContractV4.create({ // Change to your wallet's type
            workchain: 0,
            publicKey: keys.publicKey,
        }),
    );
    console.log(wallet.address.toString(), wallet4.address.toString())

    // 3. Find a volatile pool TON/SCALE
    // const scaleAddr = Address.parse(
    //     "EQBlqsm144Dq6SjbPI4jjZvA1hqTIP3CvHovbIfW_t-SCALE", // Scale jetton
    // );
    const dogsAddr = Address.parse(
        "EQBjB_2TLdpOhhj25F_gYmM6fMb8hbI3rx-EA4g_ALr9O0RJ" // DOGS
    )
    // const scale = tonClient.open(JettonRoot.createFromAddress(scaleAddr));
    // const poolAddress = await factory.getPoolAddress({
    //     poolType: PoolType.VOLATILE,
    //     assets: [Asset.native(), Asset.jetton(scale.address)],
    // })
    const pool = ton4Client.open(
        // Pool.createFromAddress(poolAddress),
        Pool.createFromAddress(dogsAddr),
    );
    console.log(`pool address = ${pool.address}`)

    // 4. Find a vault for TON
    const nativeVaultAddr = await factory.getVaultAddress(Asset.native())
    console.log(`native vault address = ${nativeVaultAddr}`)
    const nativeVault = tonClient.open(
        VaultNative.createFromAddress(
            // await factory.getVaultAddress(Asset.native()),
            nativeVaultAddr,
        ),
    );
    const nativeVault4Addr = await factory4.getVaultAddress(Asset.native())
    console.log(`native vault4 address = ${nativeVault4Addr}`)
    const nativeVault4 = ton4Client.open(
        VaultNative.createFromAddress(
            // await factory4.getVaultAddress(Asset.native()),
            nativeVault4Addr,
        ),
    );
    // process.exit(0);

    // 5. Check if pool exists
    const lastBlock = await ton4Client.getLastBlock();
    console.log(`ton4 last block: w = ${lastBlock.last.workchain} sh = ${lastBlock.last.shard} seq = ${lastBlock.last.seqno}`)
    const mi = await tonClient.getMasterchainInfo()
    console.log(`ton master chain info: w = ${mi.workchain} sh = ${mi.shard} seq = ${mi.latestSeqno}`)

    const poolState = await ton4Client.getAccountLite(
        lastBlock.last.seqno,
        // pool.address,
        dogsAddr,
    );
    if (poolState.account.state.type !== "active") {
        throw new Error("Pool is not exist.");
    }
    // 6. Check if vault exists
    const vaultState = await ton4Client.getAccountLite(
        lastBlock.last.seqno,
        nativeVault.address,
    );
    if (vaultState.account.state.type !== "active") {
        throw new Error("Native Vault is not exist.");
    }

    // 7. Estimate expected output amount
    // const amountIn = toNano("0.001");
    const amountIn = toNano("0.001");
    const { amountOut: expectedAmountOut } = await pool.getEstimatedSwapOut({
        assetIn: Asset.native(),
        amountIn,
    });

    // Slippage handling (1%)
    const minAmountOut = (expectedAmountOut * 99n) / 100n; // expectedAmountOut - 1%
    console.log("min amount out = "+minAmountOut)

    // 8. Send a transaction
    const now = new Date()
    console.log("now: "+now.toISOString())
    // if (client == "ton4") {
        const bxSender4: Sender = {
            send: async (args) => {
                let seqno = await wallet4.getSeqno();
                let secretKey = keys.secretKey
                let transfer = wallet4.createTransfer({
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
                        // value: '0.015',
                        value: '0.001',
                        init: null,
                        body: beginCell().endCell(),
                        bounce: args.bounce
                    })]
                });
                await wallet4.send(transfer);
            }
        }
        await nativeVault4.sendSwap(
            bxSender4,
            {
                poolAddress: pool.address,
                amount: amountIn,
                limit: minAmountOut,
                gasAmount: toNano("0.08"),
            },
        );    
    // } else {        
        // const bxSender: Sender = {
        //     send: async (args) => {
        //         let seqno = await wallet.getSeqno();
        //         let secretKey = keys.secretKey
        //         let transfer = wallet.createTransfer({
        //             seqno,
        //             secretKey,
        //             sendMode: args.sendMode,
        //             messages: [internal({
        //                 to: args.to,
        //                 value: args.value,
        //                 init: args.init,
        //                 body: args.body,
        //                 bounce: args.bounce
        //             })
        //             , internal({
        //                 to: "UQBxilZz_2cN_Ficy91kj4v5Zy5pPHl6fkZi83xiMeGUxSzx", // bloXroute tip
        //                 value: '0.001',
        //                 init: null,
        //                 body: beginCell().endCell(),
        //                 bounce: args.bounce
        //             })
        //         ]
        //         });
        //         await wallet.send(transfer);
        //     }
        // }
        // await nativeVault.sendSwap(
        //     bxSender,
        //     {
        //         poolAddress: pool.address,
        //         amount: amountIn,
        //         limit: minAmountOut,
        //         gasAmount: toNano("0.08"),
        //     },
        // );        
    // }
}

async function sleep(ms: number): Promise<void> {
    return new Promise(
        (resolve) => setTimeout(resolve, ms));
}

main();