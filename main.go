package main

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
)

func main() {
	receiptProof()
}

func receiptProof() {
	// txHash := common.HexToHash("0x735b5d3624a18e379888714984ef4cadd260e6584f3df8e0e622ce581ed39de2")
	// ethCli, err := ethclient.Dial("https://rpc.xlayer.tech")
	txHash := common.HexToHash("0xb9b741e52a2916b1043414605b92e6b1153c108a41cb31acb1a34719962837a6")
	ethCli, err := ethclient.Dial("https://eth.llamarpc.com")
	if err != nil {
		panic(err)
	}

	txReceipt, err := ethCli.TransactionReceipt(context.Background(), txHash)
	if err != nil {
		panic(err)
	}
	blockWithTxs, err := ethCli.BlockByNumber(context.Background(), txReceipt.BlockNumber)
	if err != nil {
		panic(err)
	}
	txs := blockWithTxs.Transactions()
	var (
		batchItems []rpc.BatchElem
	)
	for _, rawTx := range txs {
		var receipt *types.Receipt
		batchItems = append(batchItems, rpc.BatchElem{
			Method: "eth_getTransactionReceipt",
			Args: []interface{}{
				rawTx.Hash(),
			},
			Result: &receipt,
		})
	}
	err = ethCli.Client().BatchCall(batchItems)
	if err != nil {
		panic(err)
	}
	var receipts []*types.Receipt
	for _, item := range batchItems {
		receipt, ok := item.Result.(**types.Receipt)
		if !ok {
			panic("receipt convert error")
		}

		newReceipt := *receipt
		receipts = append(receipts, newReceipt)
	}

	if len(receipts) <= int(txReceipt.TransactionIndex) {
		panic(fmt.Sprintf("receipt length not enough, blockNum: %v, receipt txHash: %s", txReceipt.BlockNumber.Int64(), txHash))
	}

	var nReceipts types.Receipts
	nReceipts = receipts
	tr := trie.NewEmpty(triedb.NewDatabase(rawdb.NewMemoryDatabase(), nil))
	types.DeriveSha(nReceipts, tr)
	proof := trienode.NewProofSet()
	key, err := rlp.EncodeToBytes(txReceipt.TransactionIndex)
	if err != nil {
		panic(err)
	}
	if err = tr.Prove(key, proof); err != nil {
		panic(err)
	}

	header, err := ethCli.HeaderByNumber(context.Background(), txReceipt.BlockNumber)
	if err != nil {
		panic(err)
	}

	var receiptHash common.Hash
	copy(receiptHash[:], header.ReceiptHash[:])

	txIndex := txReceipt.TransactionIndex
	txIndexBytes := new(big.Int).SetUint64(uint64(txIndex)).Bytes()
	keyIndex, err := rlp.EncodeToBytes(txIndexBytes)
	if err != nil {
		panic(err)
	}

	_, err = trie.VerifyProof(receiptHash, keyIndex, proof)
	if err != nil {
		panic(err)
	}
}
