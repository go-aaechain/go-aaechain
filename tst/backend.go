// Copyright 2014 The go-ethereum Authors
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

// Package aae implements the aaechain protocol.
package aae

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/aaechain/go-aaechain/accounts"
	"github.com/aaechain/go-aaechain/common"
	"github.com/aaechain/go-aaechain/common/hexutil"
	"github.com/aaechain/go-aaechain/consensus"
	"github.com/aaechain/go-aaechain/consensus/clique"
	"github.com/aaechain/go-aaechain/consensus/ethash"
	"github.com/aaechain/go-aaechain/core"
	"github.com/aaechain/go-aaechain/core/bloombits"
	"github.com/aaechain/go-aaechain/core/types"
	"github.com/aaechain/go-aaechain/core/vm"
	"github.com/aaechain/go-aaechain/aae/downloader"
	"github.com/aaechain/go-aaechain/aae/filters"
	"github.com/aaechain/go-aaechain/aae/gasprice"
	"github.com/aaechain/go-aaechain/aaedb"
	"github.com/aaechain/go-aaechain/event"
	"github.com/aaechain/go-aaechain/internal/ethapi"
	"github.com/aaechain/go-aaechain/log"
	"github.com/aaechain/go-aaechain/miner"
	"github.com/aaechain/go-aaechain/node"
	"github.com/aaechain/go-aaechain/p2p"
	"github.com/aaechain/go-aaechain/params"
	"github.com/aaechain/go-aaechain/rlp"
	"github.com/aaechain/go-aaechain/rpc"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *core.ChainIndexer)
}

// aaechain implements the aaechain full node service.
type aaechain struct {
	config      *Config
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan  chan bool    // Channel for shutting down the aaeereum
	stopDbUpgrade func() error // stop chain db sequential key upgrade

	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer

	// DB interfaces
	chainDb aaedb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	ApiBackend *aaeApiBackend

	miner     *miner.Miner
	gasPrice  *big.Int
	aaeerbase common.Address

	networkId     uint64
	netRPCService *ethapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and aaeerbase)
}

func (s *aaechain) AddLesServer(ls LesServer) {
	s.lesServer = ls
	ls.SetBloomBitsIndexer(s.bloomIndexer)
}

// New creates a new aaechain object (including the
// initialisation of the common aaechain object)
func New(ctx *node.ServiceContext, config *Config) (*aaechain, error) {
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run aae.aaechain in light sync mode, use les.Lightaaechain")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err != nil {
		return nil, err
	}
	stopDbUpgrade := upgradeDeduplicateData(chainDb)
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	aae := &aaechain{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, &config.aaeash, chainConfig, chainDb),
		shutdownChan:   make(chan bool),
		stopDbUpgrade:  stopDbUpgrade,
		networkId:      config.NetworkId,
		gasPrice:       config.GasPrice,
		aaeerbase:      config.aaeerbase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks),
	}

	log.Info("Initialising aaechain protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	if !config.SkipBcVersionCheck {
		bcVersion := core.GetBlockChainVersion(chainDb)
		if bcVersion != core.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run gaae upgradedb.\n", bcVersion, core.BlockChainVersion)
		}
		core.WriteBlockChainVersion(chainDb, core.BlockChainVersion)
	}
	var (
		vmConfig    = vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
		cacheConfig = &core.CacheConfig{Disabled: config.NoPruning, TrieNodeLimit: config.TrieCache, TrieTimeLimit: config.TrieTimeout}
	)
	aae.blockchain, err = core.NewBlockChain(chainDb, cacheConfig, aae.chainConfig, aae.engine, vmConfig)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		aae.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	aae.bloomIndexer.Start(aae.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	aae.txPool = core.NewTxPool(config.TxPool, aae.chainConfig, aae.blockchain)

	if aae.protocolManager, err = NewProtocolManager(aae.chainConfig, config.SyncMode, config.NetworkId, aae.eventMux, aae.txPool, aae.engine, aae.blockchain, chainDb); err != nil {
		return nil, err
	}
	aae.miner = miner.New(aae, aae.chainConfig, aae.EventMux(), aae.engine)
	aae.miner.SetExtra(makeExtraData(config.ExtraData))

	aae.ApiBackend = &aaeApiBackend{aae, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	aae.ApiBackend.gpo = gasprice.NewOracle(aae.ApiBackend, gpoParams)

	return aae, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"gaae",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (aaedb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*aaedb.LDBDatabase); ok {
		db.Meter("aae/db/chaindata/")
	}
	return db, nil
}

// CreateConsensusEngine creates the required type of consensus engine instance for an aaechain service
func CreateConsensusEngine(ctx *node.ServiceContext, config *ethash.Config, chainConfig *params.ChainConfig, db aaedb.Database) consensus.Engine {
	// If proof-of-authority is requested, set it up
	if chainConfig.Clique != nil {
		return clique.New(chainConfig.Clique, db)
	}
	// Otherwise assume proof-of-work
	switch {
	case config.PowMode == ethash.ModeFake:
		log.Warn("aaeash used in fake mode")
		return ethash.NewFaker()
	case config.PowMode == ethash.ModeTest:
		log.Warn("aaeash used in test mode")
		return ethash.NewTester()
	case config.PowMode == ethash.ModeShared:
		log.Warn("aaeash used in shared mode")
		return ethash.NewShared()
	default:
		engine := ethash.New(ethash.Config{
			CacheDir:       ctx.ResolvePath(config.CacheDir),
			CachesInMem:    config.CachesInMem,
			CachesOnDisk:   config.CachesOnDisk,
			DatasetDir:     config.DatasetDir,
			DatasetsInMem:  config.DatasetsInMem,
			DatasetsOnDisk: config.DatasetsOnDisk,
		})
		engine.SetThreads(-1) // Disable CPU mining
		return engine
	}
}

// APIs returns the collection of RPC services the aaeereum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *aaechain) APIs() []rpc.API {
	apis := ethapi.GetAPIs(s.ApiBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "aae",
			Version:   "1.0",
			Service:   NewPublicaaechainAPI(s),
			Public:    true,
		}, {
			Namespace: "aae",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "aae",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "aae",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *aaechain) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *aaechain) aaeerbase() (eb common.Address, err error) {
	s.lock.RLock()
	aaeerbase := s.aaeerbase
	s.lock.RUnlock()

	if aaeerbase != (common.Address{}) {
		return aaeerbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			aaeerbase := accounts[0].Address

			s.lock.Lock()
			s.aaeerbase = aaeerbase
			s.lock.Unlock()

			log.Info("aaeerbase automatically configured", "address", aaeerbase)
			return aaeerbase, nil
		}
	}
	return common.Address{}, fmt.Errorf("aaeerbase must be explicitly specified")
}

// set in js console via admin interface or wrapper from cli flags
func (self *aaechain) Setaaeerbase(aaeerbase common.Address) {
	self.lock.Lock()
	self.aaeerbase = aaeerbase
	self.lock.Unlock()

	self.miner.Setaaeerbase(aaeerbase)
}

func (s *aaechain) StartMining(local bool) error {
	eb, err := s.aaeerbase()
	if err != nil {
		log.Error("Cannot start mining without aaeerbase", "err", err)
		return fmt.Errorf("aaeerbase missing: %v", err)
	}
	if clique, ok := s.engine.(*clique.Clique); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("aaeerbase account unavailable locally", "err", err)
			return fmt.Errorf("signer missing: %v", err)
		}
		clique.Authorize(eb, wallet.SignHash)
	}
	if local {
		// If local (CPU) mining is started, we can disable the transaction rejection
		// mechanism introduced to speed sync times. CPU mining on mainnet is ludicrous
		// so noone will ever hit this path, whereas marking sync done on CPU mining
		// will ensure that private networks work in single miner mode too.
		atomic.StoreUint32(&s.protocolManager.acceptTxs, 1)
	}
	go s.miner.Start(eb)
	return nil
}

func (s *aaechain) StopMining()         { s.miner.Stop() }
func (s *aaechain) IsMining() bool      { return s.miner.Mining() }
func (s *aaechain) Miner() *miner.Miner { return s.miner }

func (s *aaechain) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *aaechain) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *aaechain) TxPool() *core.TxPool               { return s.txPool }
func (s *aaechain) EventMux() *event.TypeMux           { return s.eventMux }
func (s *aaechain) Engine() consensus.Engine           { return s.engine }
func (s *aaechain) ChainDb() aaedb.Database            { return s.chainDb }
func (s *aaechain) IsListening() bool                  { return true } // Always listening
func (s *aaechain) aaeVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *aaechain) NetVersion() uint64                 { return s.networkId }
func (s *aaechain) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *aaechain) Protocols() []p2p.Protocol {
	if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	}
	return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
}

// Start implements node.Service, starting all internal goroutines needed by the
// aaechain protocol implementation.
func (s *aaechain) Start(srvr *p2p.Server) error {
	// Start the bloom bits servicing goroutines
	s.startBloomHandlers()

	// Start the RPC service
	s.netRPCService = ethapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		if s.config.LightPeers >= srvr.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", s.config.LightPeers, srvr.MaxPeers)
		}
		maxPeers -= s.config.LightPeers
	}
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// aaechain protocol.
func (s *aaechain) Stop() error {
	if s.stopDbUpgrade != nil {
		s.stopDbUpgrade()
	}
	s.bloomIndexer.Close()
	s.blockchain.Stop()
	s.protocolManager.Stop()
	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	s.txPool.Stop()
	s.miner.Stop()
	s.eventMux.Stop()

	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
