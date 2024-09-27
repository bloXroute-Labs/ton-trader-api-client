import { internal, external, beginCell, storeMessage } from "@ton/core"
import { TonClient, toNano, WalletContractV4 } from "@ton/ton";
import { mnemonicToPrivateKey } from "@ton/crypto";
import { DEX, pTON } from "@ston-fi/sdk";

async function main() {
    if (!process.env.MNEMONIC) {
        throw new Error("Environment variable MNEMONIC is required.");
    }
    const mnemonic = process.env.MNEMONIC.split(" ");
    if (!process.env.WALLET_ADDR) {
        throw new Error("Environment variable WALLET_ADDR is required.");
    }
    const walletAddr = process.env.WALLET_ADDR;
    if (!process.env.BX_TIP_ADDR) {
        throw new Error("Environment variable BX_TIP_ADDR is required.");
    }
    const bxTipAddr = process.env.BX_TIP_ADDR;

    const client = new TonClient({
        endpoint: "https://toncenter.com/api/v2/jsonRPC",
    });
    const keyPair = await mnemonicToPrivateKey(mnemonic);

    const workchain = 0;
    const wallet = WalletContractV4.create({ // Change to your wallet's type
        workchain,
        publicKey: keyPair.publicKey,
    });
    const contract = client.open(wallet);

    const router = client.open(new DEX.v1.Router());

    // swap 1 TON to STON but not less than 1 nano STON
    const txParams = await router.getSwapTonToJettonTxParams({
        userWalletAddress: walletAddr,
        proxyTon: new pTON.v1(),
        offerAmount: toNano("0.006"),
        askJettonAddress: "EQA2kCVNwVsil2EM2mB0SkXytxCqQjS4mttjDpnXmwG9T6bO", // STON
        minAskAmount: "3000000",
        queryId: 12345,
    });
    console.log(txParams)

    const seqNo = await contract.getSeqno()
    const transfer = contract.createTransfer({
        seqno: seqNo,
        secretKey: keyPair.secretKey,
        messages: [internal(txParams),
        internal({
            to: bxTipAddr, // bloXroute tip
            value: '0.005',
            init: null,
            body: beginCell().endCell(),
            bounce: true
        })],
    })

    const extMessage = external({
        to: walletAddr,
        body: transfer
    })
    const bocMessage = beginCell().store(storeMessage(extMessage)).endCell().toBoc()
    const bocEncoded = bocMessage.toString('base64')
    console.log(bocEncoded) // Send this payload to bloXroute endpoint /api/v2/submit
}

main();