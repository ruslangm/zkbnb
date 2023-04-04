/*
 * Copyright © 2021 ZkBNB Protocol
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sender

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/bnb-chain/zkbnb/dao/tx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/shopspring/decimal"
	"gorm.io/plugin/dbresolver"
	"math"
	"math/big"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	zkbnb "github.com/bnb-chain/zkbnb-eth-rpc/core"
	"github.com/bnb-chain/zkbnb-eth-rpc/rpc"
	"github.com/bnb-chain/zkbnb/common/chain"
	"github.com/bnb-chain/zkbnb/common/prove"
	"github.com/bnb-chain/zkbnb/dao/block"
	"github.com/bnb-chain/zkbnb/dao/compressedblock"
	"github.com/bnb-chain/zkbnb/dao/l1rolluptx"
	"github.com/bnb-chain/zkbnb/dao/proof"
	"github.com/bnb-chain/zkbnb/dao/sysconfig"
	sconfig "github.com/bnb-chain/zkbnb/service/sender/config"
	"github.com/bnb-chain/zkbnb/types"
)

var (
	l2BlockCommitToChainHeightMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "zkbnb",
		Name:      "l2Block_commit_to_chain_height",
		Help:      "l2Block_roll_up_height metrics.",
	})

	l2BlockCommitConfirmByChainHeightMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "zkbnb",
		Name:      "l2Block_commit_confirm_by_chain_height",
		Help:      "l2Block_roll_up_height metrics.",
	})

	l2BlockSubmitToVerifyHeightMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "zkbnb",
		Name:      "l2Block_submit_to_verify_height",
		Help:      "l2Block_roll_up_height metrics.",
	})

	l2BlockVerifiedHeightMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "zkbnb",
		Name:      "l2Block_verified_height",
		Help:      "l2Block_roll_up_height metrics.",
	})
	l2MaxWaitingTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "zkbnb",
		Name:      "l2Block_max_waiting_time",
		Help:      "l2Block_roll_up_time metrics.",
	})
	l1HeightSenderMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "zkbnb",
		Name:      "l1Block_block_height_send",
		Help:      "l1Block_block_height_send metrics.",
	})
	l1ExceptionSenderMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "zkbnb",
		Name:      "l1_Exception_send",
		Help:      "l1_Exception_send metrics.",
	})
	commitExceptionHeightMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "zkbnb",
		Name:      "commit_Exception_height",
		Help:      "commit_Exception_height metrics.",
	})
	verifyExceptionHeightMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "zkbnb",
		Name:      "verify_Exception_height",
		Help:      "verify_Exception_height metrics.",
	})
	contractBalanceMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "zkbnb",
		Name:      "contract_balance",
		Help:      "contract_balance metrics.",
	})
)

type Sender struct {
	config sconfig.Config

	// Client
	client             *rpc.ProviderClient
	kmsClient          *kms.Client
	commitAuthClient   *rpc.AuthClient
	verifyAuthClient   *rpc.AuthClient
	commitKmsKeyClient *rpc.KMSKeyClient
	verifyKmsKeyClient *rpc.KMSKeyClient

	zkbnbClient *zkbnb.ZkBNBClient

	// Data access objects
	db                   *gorm.DB
	blockModel           block.BlockModel
	compressedBlockModel compressedblock.CompressedBlockModel
	l1RollupTxModel      l1rolluptx.L1RollupTxModel
	sysConfigModel       sysconfig.SysConfigModel
	proofModel           proof.ProofModel
	txModel              tx.TxModel
}

func NewSender(c sconfig.Config) *Sender {

	masterDataSource := c.Postgres.MasterDataSource
	slaveDataSource := c.Postgres.SlaveDataSource
	db, err := gorm.Open(postgres.Open(masterDataSource))
	db.Use(dbresolver.Register(dbresolver.Config{
		Sources:  []gorm.Dialector{postgres.Open(masterDataSource)},
		Replicas: []gorm.Dialector{postgres.Open(slaveDataSource)},
	}))
	if c.ChainConfig.MaxGasPriceIncreasePercentage == 0 {
		// Calculation Formula:Percentage = ((MaxGasPrice-GasPrice)/GasPrice)*100
		c.ChainConfig.MaxGasPriceIncreasePercentage = 50
	}

	s := &Sender{
		config:               c,
		db:                   db,
		blockModel:           block.NewBlockModel(db),
		compressedBlockModel: compressedblock.NewCompressedBlockModel(db),
		l1RollupTxModel:      l1rolluptx.NewL1RollupTxModel(db),
		sysConfigModel:       sysconfig.NewSysConfigModel(db),
		proofModel:           proof.NewProofModel(db),
		txModel:              tx.NewTxModel(db),
	}

	l1RPCEndpoint, err := s.sysConfigModel.GetSysConfigByName(c.ChainConfig.NetworkRPCSysConfigName)
	if err != nil {
		logx.Severef("fatal error, failed to get network rpc configuration, err:%v, SysConfigName:%s",
			err, c.ChainConfig.NetworkRPCSysConfigName)
		panic("failed to get network rpc configuration, err:" + err.Error() + ", SysConfigName:" +
			c.ChainConfig.NetworkRPCSysConfigName)
	}
	rollupAddress, err := s.sysConfigModel.GetSysConfigByName(types.ZkBNBContract)
	if err != nil {
		logx.Severef("fatal error, failed to get zkBNB contract configuration, err:%v, SysConfigName:%s",
			err, types.ZkBNBContract)
		panic("fatal error, failed to get zkBNB contract configuration, err:" + err.Error() + "SysConfigName:" +
			types.ZkBNBContract)
	}

	s.client, err = rpc.NewClient(l1RPCEndpoint.Value)
	if err != nil {
		logx.Severef("failed to create client instance, %v", err)
		panic("failed to create client instance, err:" + err.Error())
	}

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		logx.Severef("failed to load KMS client config, %v", err)
		panic("failed to load KMS client config, err:" + err.Error())
	}
	s.kmsClient = kms.NewFromConfig(cfg)

	chainId, err := s.client.ChainID(context.Background())
	if err != nil {
		logx.Severef("fatal error, failed to get the chainId from the l1 server, err:%v", err)
		panic("fatal error, failed to get the chainId from the l1 server, err:" + err.Error())
	}

	sendSignatureMode := c.ChainConfig.SendSignatureMode
	if len(sendSignatureMode) == 0 || sendSignatureMode == sconfig.PrivateKeySignMode {
		s.commitAuthClient, err = rpc.NewAuthClient(c.AuthConfig.CommitBlockSk, chainId)
		if err != nil {
			logx.Severef("fatal error, failed to initiate commit authClient instance, err:%v", err)
			panic("fatal error, failed to initiate commit authClient instance, err:" + err.Error())
		}

		s.verifyAuthClient, err = rpc.NewAuthClient(c.AuthConfig.VerifyBlockSk, chainId)
		if err != nil {
			logx.Severef("fatal error, failed to initiate verify authClient instance, err:%v", err)
			panic("fatal error, failed to initiate verify authClient instance, err:" + err.Error())
		}
	} else if sendSignatureMode == sconfig.KeyManageSignMode {
		s.commitKmsKeyClient, err = rpc.NewKMSKeyClient(s.kmsClient, c.KMSConfig.CommitKeyId, chainId)
		if err != nil {
			logx.Severef("fatal error, failed to initiate commit kmsKeyClient instance, err:%v", err)
			panic("fatal error, failed to initiate commit kmsKeyClient instance, err:" + err.Error())
		}

		s.verifyKmsKeyClient, err = rpc.NewKMSKeyClient(s.kmsClient, c.KMSConfig.VerifyKeyId, chainId)
		if err != nil {
			logx.Severef("fatal error, failed to initiate verify kmsKeyClient instance, err:%v", err)
			panic("fatal error, failed to initiate verify kmsKeyClient instance, err:" + err.Error())
		}
	} else {
		logx.Severef("fatal error, sendSignatureMode can only be PrivateKeySignMode or KeyManageSignMode!")
		panic("fatal error, sendSignatureMode can only be PrivateKeySignMode or KeyManageSignMode!")
	}

	commitConstructor, err := s.GenerateConstructorForCommit()
	if err != nil {
		logx.Severef("fatal error, GenerateConstructorForCommit raises error:%v", err)
		panic("fatal error, GenerateConstructorForCommit raises error:" + err.Error())
	}
	verifyConstructor, err := s.GenerateConstructorForVerifyAndExecute()
	if err != nil {
		logx.Severef("fatal error, GenerateConstructorForVerifyAndExecute raises error:%v", err)
		panic("fatal error, GenerateConstructorForVerifyAndExecute raises error:" + err.Error())
	}

	s.zkbnbClient, err = zkbnb.NewZkBNBClient(s.client, rollupAddress.Value)
	s.zkbnbClient.CommitConstructor = commitConstructor
	s.zkbnbClient.VerifyConstructor = verifyConstructor
	if err != nil {
		logx.Severef("fatal error, ZkBNBClient initiate raises error:%v", err)
		panic("fatal error, ZkBNBClient initiate raises error:" + err.Error())
	}

	return s
}

func InitPrometheusFacility() {
	if err := prometheus.Register(l2BlockCommitToChainHeightMetric); err != nil {
		logx.Errorf("prometheus.Register l2BlockCommitToChainHeightMetric error: %v", err)
	}
	if err := prometheus.Register(l2BlockCommitConfirmByChainHeightMetric); err != nil {
		logx.Errorf("prometheus.Register l2BlockCommitConfirmByChainHeightMetric error: %v", err)
	}
	if err := prometheus.Register(l2BlockSubmitToVerifyHeightMetric); err != nil {
		logx.Errorf("prometheus.Register l2BlockSubmitToVerifyHeightMetric error: %v", err)
	}
	if err := prometheus.Register(l2BlockVerifiedHeightMetric); err != nil {
		logx.Errorf("prometheus.Register l2BlockVerifiedHeightMetric error: %v", err)
	}
	if err := prometheus.Register(l2BlockCommitToChainHeightMetric); err != nil {
		logx.Errorf("prometheus.Register l2BlockCommitToChainHeightMetric error: %v", err)
	}
	if err := prometheus.Register(l2BlockCommitConfirmByChainHeightMetric); err != nil {
		logx.Errorf("prometheus.Register l2BlockCommitConfirmByChainHeightMetric error: %v", err)
	}
	if err := prometheus.Register(l2BlockSubmitToVerifyHeightMetric); err != nil {
		logx.Errorf("prometheus.Register l2BlockSubmitToVerifyHeightMetric error: %v", err)
	}
	if err := prometheus.Register(l2BlockVerifiedHeightMetric); err != nil {
		logx.Errorf("prometheus.Register l2BlockVerifiedHeightMetric error: %v", err)
	}
	if err := prometheus.Register(l2MaxWaitingTimeMetric); err != nil {
		logx.Errorf("prometheus.Register l2MaxWaitingTimeMetric error: %v", err)
	}
	if err := prometheus.Register(l1HeightSenderMetric); err != nil {
		logx.Errorf("prometheus.Register l1HeightSenderMetric error: %v", err)
	}
	if err := prometheus.Register(l1ExceptionSenderMetric); err != nil {
		logx.Errorf("prometheus.Register l1ExceptionSenderMetric error: %v", err)
	}
	if err := prometheus.Register(commitExceptionHeightMetric); err != nil {
		logx.Errorf("prometheus.Register commitExceptionHeightMetric error: %v", err)
	}
	if err := prometheus.Register(verifyExceptionHeightMetric); err != nil {
		logx.Errorf("prometheus.Register verifyExceptionHeightMetric error: %v", err)
	}
	if err := prometheus.Register(contractBalanceMetric); err != nil {
		logx.Errorf("prometheus.Register contractBalanceMetric error: %v", err)
	}
}

func (s *Sender) CommitBlocks() (err error) {
	info, err := s.sysConfigModel.GetSysConfigByName("ZkBNBContract")
	if err == nil {
		balance, err := s.client.GetBalance(info.Value)
		fbalance := new(big.Float)
		fbalance.SetString(balance.String())
		ethValue := new(big.Float).Quo(fbalance, big.NewFloat(math.Pow10(18)))
		if err != nil {
			contractBalanceMetric.Set(float64(0))
		} else {
			f, _ := ethValue.Float64()
			contractBalanceMetric.Set(f)
		}
	}

	pendingTx, err := s.l1RollupTxModel.GetLatestPendingTx(l1rolluptx.TxTypeCommit)
	if err != nil && err != types.DbErrNotFound {
		return err
	}
	// No need to submit new transaction if there is any pending commit txs.
	if pendingTx != nil {
		return nil
	}
	lastHandledTx, err := s.l1RollupTxModel.GetLatestHandledTx(l1rolluptx.TxTypeCommit)
	if err != nil && err != types.DbErrNotFound {
		return err
	}
	start := int64(1)
	if lastHandledTx != nil {
		start = lastHandledTx.L2BlockHeight + 1
	}

	// commit new blocks
	blocks, err := s.GetCompressedBlocksForCommit(start)
	if err != nil {
		return err
	}

	if len(blocks) == 0 {
		return nil
	}
	pendingCommitBlocks, err := ConvertBlocksForCommitToCommitBlockInfos(blocks, s.txModel)
	if err != nil {
		return fmt.Errorf("failed to get commit block info, err: %v", err)
	}
	// get last block info
	lastStoredBlockInfo := defaultBlockHeader()
	if lastHandledTx != nil {
		lastHandledBlockInfo, err := s.blockModel.GetBlockByHeight(lastHandledTx.L2BlockHeight)
		if err != nil {
			return fmt.Errorf("failed to get block info, err: %v", err)
		}
		// construct last stored block header
		lastStoredBlockInfo = chain.ConstructStoredBlockInfo(lastHandledBlockInfo)
	}

	var gasPrice *big.Int
	if s.config.ChainConfig.GasPrice > 0 {
		gasPrice = big.NewInt(int64(s.config.ChainConfig.GasPrice))
	} else {
		gasPrice, err = s.client.SuggestGasPrice(context.Background())
		if err != nil {
			logx.Errorf("failed to fetch gas price: %v", err)
			return err
		}
	}
	var txHash string
	var nonce uint64

	maxGasPrice := (decimal.NewFromInt(gasPrice.Int64()).Mul(decimal.NewFromInt(int64(s.config.ChainConfig.MaxGasPriceIncreasePercentage))).Div(decimal.NewFromInt(100))).Add(decimal.NewFromInt(gasPrice.Int64()))
	nonce, err = s.client.GetPendingNonce(s.GetCommitAddress().Hex())
	if err != nil {
		return fmt.Errorf("failed to get nonce for commit block, errL %v", err)
	}

	l1RollupTx, err := s.l1RollupTxModel.GetLatestByNonce(int64(nonce), l1rolluptx.TxTypeCommit)
	if err != nil && err != types.DbErrNotFound {
		return fmt.Errorf("failed to get latest l1 rollup tx by nonce %d, err: %v", nonce, err)
	}
	if l1RollupTx != nil && l1RollupTx.L1Nonce == int64(nonce) {
		standByGasPrice := decimal.NewFromInt(l1RollupTx.GasPrice).Add(decimal.NewFromInt(l1RollupTx.GasPrice).Mul(decimal.NewFromFloat(0.1)))
		if standByGasPrice.GreaterThan(maxGasPrice) {
			logx.Errorf("abandon commit block to l1, gasPrice>maxGasPrice,l1 nonce: %s,gasPrice: %s,maxGasPrice: %s", nonce, standByGasPrice, maxGasPrice)
			return nil
		}
		gasPrice = standByGasPrice.RoundUp(0).BigInt()
		logx.Infof("speed up commit block to l1,l1 nonce: %s,gasPrice: %s", nonce, gasPrice)
	}
	retry := false
	for {
		if retry {
			newNonce, err := s.client.GetPendingNonce(s.GetCommitAddress().Hex())
			if err != nil {
				return fmt.Errorf("failed to get nonce for commit block, errL %v", err)
			}
			if nonce != newNonce {
				return fmt.Errorf("failed to retry for commit block, nonce=%d,newNonce=%d", nonce, newNonce)
			}
			standByGasPrice := decimal.NewFromInt(gasPrice.Int64()).Add(decimal.NewFromInt(gasPrice.Int64()).Mul(decimal.NewFromFloat(0.1)))
			if standByGasPrice.GreaterThan(maxGasPrice) {
				logx.Errorf("abandon commit block to l1, gasPrice>maxGasPrice,l1 nonce: %s,gasPrice: %s,maxGasPrice: %s", nonce, standByGasPrice, maxGasPrice)
				return nil
			}
			gasPrice = standByGasPrice.RoundUp(0).BigInt()
			logx.Infof("speed up commit block to l1,l1 nonce: %s,gasPrice: %s", nonce, gasPrice)

			// Judge whether the blocks should be committed to the chain for better gas consumption
			shouldCommit := s.ShouldCommitBlocks(lastStoredBlockInfo, pendingCommitBlocks,
				blocks, gasPrice, s.config.ChainConfig.GasLimit, nonce)
			if !shouldCommit {
				logx.Errorf("abandon commit block to l1, EstimateGas value is greater than MaxUnitGas!")
				return nil
			}

			// commit blocks on-chain
			txHash, err = s.zkbnbClient.CommitBlocksWithNonce(
				lastStoredBlockInfo,
				pendingCommitBlocks,
				gasPrice,
				s.config.ChainConfig.GasLimit, nonce)
			if err != nil {
				commitExceptionHeightMetric.Set(float64(pendingCommitBlocks[len(pendingCommitBlocks)-1].BlockNumber))
				if err.Error() == "replacement transaction underpriced" || err.Error() == "transaction underpriced" {
					logx.Errorf("failed to send commit tx,try again: errL %v:%s", err, txHash)
					retry = true
					continue
				}
				break
			}

			commitExceptionHeightMetric.Set(float64(0))
			for _, pendingCommitBlock := range pendingCommitBlocks {
				l2BlockCommitToChainHeightMetric.Set(float64(pendingCommitBlock.BlockNumber))
			}
			newRollupTx := &l1rolluptx.L1RollupTx{
				L1TxHash:      txHash,
				TxStatus:      l1rolluptx.StatusPending,
				TxType:        l1rolluptx.TxTypeCommit,
				L2BlockHeight: int64(pendingCommitBlocks[len(pendingCommitBlocks)-1].BlockNumber),
				L1Nonce:       int64(nonce),
				GasPrice:      gasPrice.Int64(),
			}
			err = s.l1RollupTxModel.CreateL1RollupTx(newRollupTx)
			if err != nil {
				return fmt.Errorf("failed to create tx in database, err: %v", err)
			}
			l2BlockCommitToChainHeightMetric.Set(float64(newRollupTx.L2BlockHeight))
			logx.Infof("new blocks have been committed(height): %v:%s", newRollupTx.L2BlockHeight, newRollupTx.L1TxHash)
			return nil
		}
	}
	return nil
}

func (s *Sender) UpdateSentTxs() (err error) {
	pendingTxs, err := s.l1RollupTxModel.GetL1RollupTxsByStatus(l1rolluptx.StatusPending)
	if err != nil {
		if err == types.DbErrNotFound {
			return nil
		}
		return fmt.Errorf("failed to get pending txs, err: %v", err)
	}
	latestL1Height, err := s.client.GetHeight()
	if err != nil {
		return fmt.Errorf("failed to get l1 block height, err: %v", err)
	}
	l1HeightSenderMetric.Set(float64(latestL1Height))
	var (
		pendingUpdateRxs         []*l1rolluptx.L1RollupTx
		pendingUpdateProofStatus = make(map[int64]int)
	)
	for _, pendingTx := range pendingTxs {
		txHash := pendingTx.L1TxHash
		receipt, err := s.client.GetTransactionReceipt(txHash)
		if err != nil {
			logx.Errorf("query transaction receipt %s failed, err: %v", txHash, err)
			if time.Now().After(pendingTx.UpdatedAt.Add(time.Duration(s.config.ChainConfig.MaxWaitingTime) * time.Second)) {
				// No need to check the response, do best effort.
				logx.Errorf("delete timeout l1 rollup tx, tx_hash=%s", pendingTx.L1TxHash)
				//nolint:errcheck
				s.l1RollupTxModel.DeleteL1RollupTx(pendingTx)
				l2MaxWaitingTimeMetric.Set(float64(pendingTx.L2BlockHeight))
			}
			continue
		}
		if receipt.Status == 0 {
			// Should direct mark tx deleted
			logx.Infof("delete timeout l1 rollup tx, tx_hash=%s", pendingTx.L1TxHash)
			//nolint:errcheck
			s.l1RollupTxModel.DeleteL1RollupTx(pendingTx)
			l1ExceptionSenderMetric.Set(float64(pendingTx.L2BlockHeight))
			// It is critical to have any failed transactions
			logx.Severef("unexpected failed tx: %v", txHash)
			panic(fmt.Sprintf("unexpected failed tx: %v", txHash))
		}
		l2MaxWaitingTimeMetric.Set(float64(0))
		l1ExceptionSenderMetric.Set(float64(0))

		// not finalized yet
		if latestL1Height < receipt.BlockNumber.Uint64()+s.config.ChainConfig.ConfirmBlocksCount {
			continue
		}
		var validTx bool
		for _, vlog := range receipt.Logs {
			switch vlog.Topics[0].Hex() {
			case zkbnbLogBlockCommitSigHash.Hex():
				var event zkbnb.ZkBNBBlockCommit
				if err = ZkBNBContractAbi.UnpackIntoInterface(&event, EventNameBlockCommit, vlog.Data); err != nil {
					return err
				}
				validTx = int64(event.BlockNumber) == pendingTx.L2BlockHeight
			case zkbnbLogBlockVerificationSigHash.Hex():
				var event zkbnb.ZkBNBBlockVerification
				if err = ZkBNBContractAbi.UnpackIntoInterface(&event, EventNameBlockVerification, vlog.Data); err != nil {
					return err
				}
				validTx = int64(event.BlockNumber) == pendingTx.L2BlockHeight
				pendingUpdateProofStatus[int64(event.BlockNumber)] = proof.Confirmed
			case zkbnbLogBlocksRevertSigHash.Hex():
				// TODO revert
			default:
			}
		}

		if validTx {
			pendingTx.TxStatus = l1rolluptx.StatusHandled
			pendingUpdateRxs = append(pendingUpdateRxs, pendingTx)
			if pendingTx.TxType == l1rolluptx.TxTypeCommit {
				l2BlockCommitConfirmByChainHeightMetric.Set(float64(pendingTx.L2BlockHeight))
			} else if pendingTx.TxType == l1rolluptx.TxTypeVerifyAndExecute {
				l2BlockVerifiedHeightMetric.Set(float64(pendingTx.L2BlockHeight))
			}
		}
	}

	//update db
	err = s.db.Transaction(func(tx *gorm.DB) error {
		//update l1 rollup txs
		err := s.l1RollupTxModel.UpdateL1RollupTxsStatusInTransact(tx, pendingUpdateRxs)
		if err != nil {
			return err
		}
		//update proof status
		err = s.proofModel.UpdateProofsInTransact(tx, pendingUpdateProofStatus)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to updte rollup txs, err:%v", err)
	}
	return nil
}

func (s *Sender) VerifyAndExecuteBlocks() (err error) {
	pendingTx, err := s.l1RollupTxModel.GetLatestPendingTx(l1rolluptx.TxTypeVerifyAndExecute)
	if err != nil && err != types.DbErrNotFound {
		return err
	}
	// No need to submit new transaction if there is any pending verification txs.
	if pendingTx != nil {
		return nil
	}

	lastHandledTx, err := s.l1RollupTxModel.GetLatestHandledTx(l1rolluptx.TxTypeVerifyAndExecute)
	if err != nil && err != types.DbErrNotFound {
		return err
	}

	start := int64(1)
	if lastHandledTx != nil {
		start = lastHandledTx.L2BlockHeight + 1
	}
	blocks, err := s.GetBlocksForVerifyAndExecute(start)
	if err != nil {
		return err
	}
	if len(blocks) == 0 {
		return nil
	}
	pendingVerifyAndExecuteBlocks, err := ConvertBlocksToVerifyAndExecuteBlockInfos(blocks)
	if err != nil {
		return fmt.Errorf("unable to convert blocks to commit block infos: %v", err)
	}

	blockProofs, err := s.proofModel.GetProofsBetween(start, start+int64(len(blocks))-1)
	if err != nil {
		if err == types.DbErrNotFound {
			return nil
		}
		return fmt.Errorf("unable to get proofs, err: %v", err)
	}
	if len(blockProofs) != len(blocks) {
		return types.AppErrRelatedProofsNotReady
	}
	// add sanity check
	for i := range blockProofs {
		if blockProofs[i].BlockNumber != blocks[i].BlockHeight {
			return types.AppErrProofNumberNotMatch
		}
	}
	var proofs []*big.Int
	for _, bProof := range blockProofs {
		var proofInfo *prove.FormattedProof
		err = json.Unmarshal([]byte(bProof.ProofInfo), &proofInfo)
		if err != nil {
			return err
		}
		proofs = append(proofs, proofInfo.A[:]...)
		proofs = append(proofs, proofInfo.B[0][0], proofInfo.B[0][1])
		proofs = append(proofs, proofInfo.B[1][0], proofInfo.B[1][1])
		proofs = append(proofs, proofInfo.C[:]...)
	}

	var gasPrice *big.Int
	if s.config.ChainConfig.GasPrice > 0 {
		gasPrice = big.NewInt(int64(s.config.ChainConfig.GasPrice))
	} else {
		gasPrice, err = s.client.SuggestGasPrice(context.Background())
		if err != nil {
			logx.Errorf("failed to fetch gas price: %v", err)
			return err
		}
	}

	var txHash string
	var nonce uint64

	maxGasPrice := (decimal.NewFromInt(gasPrice.Int64()).Mul(decimal.NewFromInt(int64(s.config.ChainConfig.MaxGasPriceIncreasePercentage))).Div(decimal.NewFromInt(100))).Add(decimal.NewFromInt(gasPrice.Int64()))
	nonce, err = s.client.GetPendingNonce(s.GetVerifyAddress().Hex())
	if err != nil {
		return fmt.Errorf("failed to get nonce for commit block, errL %v", err)
	}

	l1RollupTx, err := s.l1RollupTxModel.GetLatestByNonce(int64(nonce), l1rolluptx.TxTypeVerifyAndExecute)
	if err != nil && err != types.DbErrNotFound {
		return fmt.Errorf("failed to get latest l1 rollup tx by nonce %d, err: %v", nonce, err)
	}
	if l1RollupTx != nil && l1RollupTx.L1Nonce == int64(nonce) {
		standByGasPrice := decimal.NewFromInt(l1RollupTx.GasPrice).Add(decimal.NewFromInt(l1RollupTx.GasPrice).Mul(decimal.NewFromFloat(0.1)))
		if standByGasPrice.GreaterThan(maxGasPrice) {
			logx.Errorf("abandon verify block to l1, gasPrice>maxGasPrice,l1 nonce: %s,gasPrice: %s,maxGasPrice: %s", nonce, standByGasPrice, maxGasPrice)
			return nil
		}
		gasPrice = standByGasPrice.RoundUp(0).BigInt()
		logx.Infof("speed up verify block to l1,l1 nonce: %s,gasPrice: %s", nonce, gasPrice)
	}
	retry := false
	for {
		if retry {
			newNonce, err := s.client.GetPendingNonce(s.GetVerifyAddress().Hex())
			if err != nil {
				return fmt.Errorf("failed to get nonce for verify block, errL %v", err)
			}
			if nonce != newNonce {
				return fmt.Errorf("failed to retry for verify block, nonce=%d,newNonce=%d", nonce, newNonce)
			}
			standByGasPrice := decimal.NewFromInt(gasPrice.Int64()).Add(decimal.NewFromInt(gasPrice.Int64()).Mul(decimal.NewFromFloat(0.1)))
			if standByGasPrice.GreaterThan(maxGasPrice) {
				logx.Errorf("abandon verify block to l1, gasPrice>maxGasPrice,l1 nonce: %s,gasPrice: %s,maxGasPrice: %s", nonce, standByGasPrice, maxGasPrice)
				return nil
			}
			gasPrice = standByGasPrice.RoundUp(0).BigInt()
			logx.Infof("speed up verify block to l1,l1 nonce: %s,gasPrice: %s", nonce, gasPrice)
		}

		// Judge whether the blocks should be verified and executed to the chain for better gas consumption
		shouldVerifyAndExecute := s.ShouldVerifyAndExecuteBlocks(blocks, pendingVerifyAndExecuteBlocks, proofs,
			gasPrice, s.config.ChainConfig.GasLimit, nonce)
		if !shouldVerifyAndExecute {
			logx.Errorf("abandon verify and execute block to l1, EstimateGas value is greater than MaxUnitGas!")
			return nil
		}

		// Verify blocks on-chain
		txHash, err = s.zkbnbClient.VerifyAndExecuteBlocksWithNonce(
			pendingVerifyAndExecuteBlocks,
			proofs, gasPrice, s.config.ChainConfig.GasLimit, nonce)
		if err != nil {
			verifyExceptionHeightMetric.Set(float64(pendingVerifyAndExecuteBlocks[len(pendingVerifyAndExecuteBlocks)-1].BlockHeader.BlockNumber))
			if err.Error() == "replacement transaction underpriced" || err.Error() == "transaction underpriced" {
				logx.Errorf("failed to send verify tx,try again: errL %v:%s", err, txHash)
				retry = true
				continue
			}
			return fmt.Errorf("failed to send verify tx: %v:%s", err, txHash)
		}
		break
	}

	verifyExceptionHeightMetric.Set(float64(0))
	for _, pendingVerifyAndExecuteBlock := range pendingVerifyAndExecuteBlocks {
		l2BlockSubmitToVerifyHeightMetric.Set(float64(pendingVerifyAndExecuteBlock.BlockHeader.BlockNumber))
	}
	newRollupTx := &l1rolluptx.L1RollupTx{
		L1TxHash:      txHash,
		TxStatus:      l1rolluptx.StatusPending,
		TxType:        l1rolluptx.TxTypeVerifyAndExecute,
		L2BlockHeight: int64(pendingVerifyAndExecuteBlocks[len(pendingVerifyAndExecuteBlocks)-1].BlockHeader.BlockNumber),
		L1Nonce:       int64(nonce),
		GasPrice:      gasPrice.Int64(),
	}
	err = s.l1RollupTxModel.CreateL1RollupTx(newRollupTx)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("failed to create rollup tx in db %v", err))
	}
	l2BlockSubmitToVerifyHeightMetric.Set(float64(newRollupTx.L2BlockHeight))
	logx.Infof("new blocks have been verified and executed(height): %d:%s", newRollupTx.L2BlockHeight, newRollupTx.L1TxHash)
	return nil
}

func (s *Sender) GetCompressedBlocksForCommit(start int64) (blocksForCommit []*compressedblock.CompressedBlock, err error) {
	commitTxCountLimit := sconfig.GetSenderConfig().CommitTxCountLimit
	maxCommitBlockCount := sconfig.GetSenderConfig().MaxCommitBlockCount
	var totalTxCount uint64 = 0
	for {
		blocks, err := s.compressedBlockModel.GetCompressedBlocksBetween(start,
			start+int64(maxCommitBlockCount))
		if err != nil && err != types.DbErrNotFound {
			return nil, fmt.Errorf("failed to get compress block err: %v", err)
		}

		totalTxCount = s.CalculateTotalTxCountForCompressBlock(blocks)
		if totalTxCount < commitTxCountLimit {
			return blocks, nil
		}
		if maxCommitBlockCount > 1 {
			maxCommitBlockCount--
		}
	}
}

func (s *Sender) ShouldCommitBlocks(lastBlock zkbnb.StorageStoredBlockInfo,
	commitBlocksInfo []zkbnb.ZkBNBCommitBlockInfo, blocks []*compressedblock.CompressedBlock,
	gasPrice *big.Int, gasLimit uint64, nonce uint64) bool {

	// Judge the tx count waiting to be committed, if the tx count is greater
	// than the maxCommitTxCount, commit the blocks directly
	maxCommitTxCount := sconfig.GetSenderConfig().MaxCommitTxCount
	totalTxCount := s.CalculateTotalTxCountForCompressBlock(blocks)
	if totalTxCount > maxCommitTxCount {
		return true
	}

	// Judge the time interval of the block waiting to be committed, if the time interval is greater
	// than the maxCommitBlockInterval, commit the blocks directly
	maxCommitBlockInterval := sconfig.GetSenderConfig().MaxCommitBlockInterval
	commitBlockInterval := s.CalculateBlockIntervalForCompressedBlock(blocks)
	if commitBlockInterval > int64(maxCommitBlockInterval) {
		return true
	}

	// Judge the average tx gas consumption for the committing operation, if the average tx gas consumption is greater
	// than the maxCommitAvgUnitGas, abandon commit operation for temporary
	estimatedFee, err := s.zkbnbClient.EstimateCommitGasWithNonce(lastBlock, commitBlocksInfo, gasPrice, gasLimit, nonce)
	if err != nil {
		logx.Errorf("abandon commit block to l1, EstimateGas operation get some error:%s", err.Error())
		return false
	}

	maxCommitAvgUnitGas := sconfig.GetSenderConfig().MaxCommitAvgUnitGas
	unitGas := estimatedFee / totalTxCount
	if unitGas <= maxCommitAvgUnitGas {
		logx.Info("abandon commit block to l1, UnitGasFee is greater than MaxCommitBlockUnitGas, UnitGasFee:%d, "+
			"MaxCommitAvgUnitGas:%d", unitGas, maxCommitAvgUnitGas)
		return true
	}
	return false
}

func (s *Sender) ShouldVerifyAndExecuteBlocks(blocks []*block.Block, verifyAndExecuteBlocksInfo []zkbnb.ZkBNBVerifyAndExecuteBlockInfo,
	proofs []*big.Int, gasPrice *big.Int, gasLimit uint64, nonce uint64) bool {

	// Judge the tx count waiting to be verified and executed, if the tx count is greater
	// than the maxVerifyTxCount, verify and execute the blocks directly
	maxVerifyTxCount := sconfig.GetSenderConfig().MaxVerifyTxCount
	totalTxCount := s.CalculateTotalTxCountForBlock(blocks)
	if totalTxCount > maxVerifyTxCount {
		return true
	}

	// Judge the time interval of the block waiting to be verified and executed, if the time interval is greater
	// than the maxVerifyBlockInterval, verify and execute the blocks directly
	maxVerifyBlockInterval := sconfig.GetSenderConfig().MaxVerifyBlockInterval
	verifyBlockInterval := s.CalculateBlockIntervalForBlock(blocks)
	if verifyBlockInterval > int64(maxVerifyBlockInterval) {
		return true
	}

	// Judge the average tx gas consumption for the verifying and executing operation, if the average tx gas consumption is greater
	// than the maxVerifyAvgUnitGas, abandon verify and execute operation for temporary
	estimatedFee, err := s.zkbnbClient.EstimateVerifyAndExecuteWithNonce(verifyAndExecuteBlocksInfo, proofs, gasPrice, gasLimit, nonce)
	if err != nil {
		logx.Errorf("abandon commit block to l1, EstimateGas operation get some error:%s", err.Error())
		return false
	}

	maxVerifyAvgUnitGas := sconfig.GetSenderConfig().MaxVerifyAvgUnitGas
	unitGas := estimatedFee / totalTxCount
	if unitGas > maxVerifyAvgUnitGas {
		logx.Info("abandon verify and execute block to l1, UnitGasFee is greater than maxVerifyAvgUnitGas, UnitGasFee:%d, "+
			"MaxVerifyAvgUnitGas:%d", unitGas, maxVerifyAvgUnitGas)
		return false
	}
	return true
}

func (s *Sender) GetBlocksForVerifyAndExecute(start int64) (blocks []*block.Block, err error) {
	verifyTxCountLimit := sconfig.GetSenderConfig().VerifyTxCountLimit
	maxVerifyBlockCount := sconfig.GetSenderConfig().MaxVerifyBlockCount
	var totalTxCount uint64 = 0
	for {
		blocks, err := s.blockModel.GetCommittedBlocksBetween(start,
			start+int64(maxVerifyBlockCount))
		if err != nil && err != types.DbErrNotFound {
			return nil, fmt.Errorf("unable to get blocks to prove, err: %v", err)
		}

		totalTxCount = s.CalculateTotalTxCountForBlock(blocks)
		if totalTxCount < verifyTxCountLimit {
			return blocks, nil
		}
		if maxVerifyBlockCount > 1 {
			maxVerifyBlockCount--
		}
	}
}

func (s *Sender) CalculateBlockIntervalForCompressedBlock(blocks []*compressedblock.CompressedBlock) int64 {
	if len(blocks) > 0 {
		block := blocks[0]
		interval := time.Now().Unix() - block.CreatedAt.Unix()
		return interval
	}
	return 0
}

func (s *Sender) CalculateBlockIntervalForBlock(blocks []*block.Block) int64 {
	if len(blocks) > 0 {
		block := blocks[0]
		interval := time.Now().Unix() - block.CreatedAt.Unix()
		return interval
	}
	return 0
}

func (s *Sender) CalculateTotalTxCountForCompressBlock(blocks []*compressedblock.CompressedBlock) uint64 {
	var totalTxCount uint16 = 0
	if len(blocks) > 0 {
		for _, b := range blocks {
			totalTxCount = totalTxCount + b.RealBlockSize
		}
		return uint64(totalTxCount)
	}
	return 0
}

func (s *Sender) CalculateTotalTxCountForBlock(blocks []*block.Block) uint64 {
	var totalTxCount uint16 = 0
	if len(blocks) > 0 {
		for _, b := range blocks {
			totalTxCount = totalTxCount + b.BlockSize
		}
		return uint64(totalTxCount)
	}
	return 0
}

func (s *Sender) GenerateConstructorForCommit() (zkbnb.TransactOptsConstructor, error) {
	sendSignatureMode := s.config.ChainConfig.SendSignatureMode
	if len(sendSignatureMode) == 0 || sendSignatureMode == sconfig.PrivateKeySignMode {
		return s.commitAuthClient, nil
	} else if sendSignatureMode == sconfig.KeyManageSignMode {
		return s.commitKmsKeyClient, nil
	}
	return nil, errors.New("sendSignatureMode can only be PrivateKeySignMode or KeyManageSignMode")
}

func (s *Sender) GenerateConstructorForVerifyAndExecute() (zkbnb.TransactOptsConstructor, error) {
	sendSignatureMode := s.config.ChainConfig.SendSignatureMode
	if len(sendSignatureMode) == 0 || sendSignatureMode == sconfig.PrivateKeySignMode {
		return s.verifyAuthClient, nil
	} else if sendSignatureMode == sconfig.KeyManageSignMode {
		return s.verifyKmsKeyClient, nil
	}
	return nil, errors.New("sendSignatureMode can only be PrivateKeySignMode or KeyManageSignMode")
}

func (s *Sender) GetCommitAddress() common.Address {
	sendSignatureMode := s.config.ChainConfig.SendSignatureMode
	if len(sendSignatureMode) == 0 || sendSignatureMode == sconfig.PrivateKeySignMode {
		return s.commitAuthClient.GetL1Address()
	} else if sendSignatureMode == sconfig.KeyManageSignMode {
		return s.commitKmsKeyClient.GetL1Address()
	}
	return [20]byte{}
}

func (s *Sender) GetVerifyAddress() common.Address {
	sendSignatureMode := s.config.ChainConfig.SendSignatureMode
	if len(sendSignatureMode) == 0 || sendSignatureMode == sconfig.PrivateKeySignMode {
		return s.verifyAuthClient.GetL1Address()
	} else if sendSignatureMode == sconfig.KeyManageSignMode {
		return s.verifyKmsKeyClient.GetL1Address()
	}
	return [20]byte{}
}

func (s *Sender) Shutdown() {
	sqlDB, err := s.db.DB()
	if err == nil && sqlDB != nil {
		err = sqlDB.Close()
	}
	if err != nil {
		logx.Errorf("close db error: %s", err.Error())
	}
}
