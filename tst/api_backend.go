// Copyright 2015 The go-ethereum Authors
// This file is part of the go-aaeereum library.
//
// The go-aaeereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-aaeereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-aaeereum library. If not, see <http://www.gnu.org/licenses/>.

package aae

import (
	"context"
	"math/big"

	"github.com/aaechain/go-aaechain/accounts"
	"github.com/aaechain/go-aaechain/common"
	"github.com/aaechain/go-aaechain/common/math"
	"github.com/aaechain/go-aaechain/core"
	"github.com/aaechain/go-aaechain/core/bloombits"
	"github.com/aaechain/go-aaechain/core/state"
	"github.com/aaechain/go-aaechain/core/types"
	"github.com/aaechain/go-aaechain/core/vm"
	"github.com/aaechain/go-aaechain/aae/downloader"
	"github.com/aaechain/go-aaechain/aae/gasprice"
	"github.com/aaechain/go-aaechain/aaedb"
	"github.com/aaechain/go-aaechain/event"
	"github.com/aaechain/go-aaechain/params"
	"github.com/aaechain/go-aaechain/rpc"
)

// aaeApiBackend implements ethapi.Backend for full nodes
type aaeApiBackend struct {
	aae *aaechain
	gpo *gasprice.Oracle
}

func (b *aaeApiBackend) ChainConfig() *params.ChainConfig {
	return b.aae.chainConfig
}

func (b *aaeApiBackend) CurrentBlock() *types.Block {
	return b.aae.blockchain.CurrentBlock()
}

func (b *aaeApiBackend) SetHead(number uint64) {
	b.aae.protocolManager.downloader.Cancel()
	b.aae.blockchain.SetHead(number)
}

func (b *aaeApiBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.aae.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.aae.blockchain.CurrentBlock().Header(), nil
	}
	return b.aae.blockchain.GetHeaderByNumber(uint64(blockNr)), nil
}

func (b *aaeApiBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.aae.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.aae.blockchain.CurrentBlock(), nil
	}
	return b.aae.blockchain.GetBlockByNumber(uint64(blockNr)), nil
}

func (b *aaeApiBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block, state := b.aae.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := b.aae.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *aaeApiBackend) GetBlock(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	return b.aae.blockchain.GetBlockByHash(blockHash), nil
}

func (b *aaeApiBackend) GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	return core.GetBlockReceipts(b.aae.chainDb, blockHash, core.GetBlockNumber(b.aae.chainDb, blockHash)), nil
}

func (b *aaeApiBackend) GetLogs(ctx context.Context, blockHash common.Hash) ([][]*types.Log, error) {
	receipts := core.GetBlockReceipts(b.aae.chainDb, blockHash, core.GetBlockNumber(b.aae.chainDb, blockHash))
	if receipts == nil {
		return nil, nil
	}
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs, nil
}

func (b *aaeApiBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.aae.blockchain.GetTdByHash(blockHash)
}

func (b *aaeApiBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	vmError := func() error { return nil }

	context := core.NewEVMContext(msg, header, b.aae.BlockChain(), nil)
	return vm.NewEVM(context, state, b.aae.chainConfig, vmCfg), vmError, nil
}

func (b *aaeApiBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.aae.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *aaeApiBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.aae.BlockChain().SubscribeChainEvent(ch)
}

func (b *aaeApiBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.aae.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *aaeApiBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.aae.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *aaeApiBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.aae.BlockChain().SubscribeLogsEvent(ch)
}

func (b *aaeApiBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.aae.txPool.AddLocal(signedTx)
}

func (b *aaeApiBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.aae.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *aaeApiBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.aae.txPool.Get(hash)
}

func (b *aaeApiBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.aae.txPool.State().GetNonce(addr), nil
}

func (b *aaeApiBackend) Stats() (pending int, queued int) {
	return b.aae.txPool.Stats()
}

func (b *aaeApiBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.aae.TxPool().Content()
}

func (b *aaeApiBackend) SubscribeTxPreEvent(ch chan<- core.TxPreEvent) event.Subscription {
	return b.aae.TxPool().SubscribeTxPreEvent(ch)
}

func (b *aaeApiBackend) Downloader() *downloader.Downloader {
	return b.aae.Downloader()
}

func (b *aaeApiBackend) ProtocolVersion() int {
	return b.aae.aaeVersion()
}

func (b *aaeApiBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *aaeApiBackend) ChainDb() aaedb.Database {
	return b.aae.ChainDb()
}

func (b *aaeApiBackend) EventMux() *event.TypeMux {
	return b.aae.EventMux()
}

func (b *aaeApiBackend) AccountManager() *accounts.Manager {
	return b.aae.AccountManager()
}

func (b *aaeApiBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.aae.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *aaeApiBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.aae.bloomRequests)
	}
}
