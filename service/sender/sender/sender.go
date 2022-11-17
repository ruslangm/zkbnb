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
	"github.com/prometheus/client_golang/prometheus"
	"math/big"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

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
	commitLatestHandledMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "zkbnb",
		Name:      "commit_latest_handled",
		Help:      "commit latest handled height metrics.",
	})
	commitOperationMetrics = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "zkbnb",
		Name:      "commit_operation_time",
		Help:      "commit operation time metrics.",
	})
	verifyLatestHandledMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "zkbnb",
		Name:      "verify_latest_handled",
		Help:      "verify latest handled height metrics.",
	})
	verifyOperationMetrics = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "zkbnb",
		Name:      "verify_operation_time",
		Help:      "verify operation time metrics.",
	})
	sendPendingMetrics = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "zkbnb",
		Name:      "send_pending_count",
		Help:      "send pending count metrics.",
	})
	sendOperationMetrics = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "zkbnb",
		Name:      "send_operation_time",
		Help:      "send operation time metrics.",
	})
)

type Sender struct {
	config sconfig.Config

	// Client
	cli           *rpc.ProviderClient
	authCli       *rpc.AuthClient
	zkbnbInstance *zkbnb.ZkBNB

	// Data access objects
	db                   *gorm.DB
	blockModel           block.BlockModel
	compressedBlockModel compressedblock.CompressedBlockModel
	l1RollupTxModel      l1rolluptx.L1RollupTxModel
	sysConfigModel       sysconfig.SysConfigModel
	proofModel           proof.ProofModel
}

func NewSender(c sconfig.Config) *Sender {
	if err := prometheus.Register(commitLatestHandledMetric); err != nil {
		logx.Errorf("prometheus.Register commitLatestHandledMetric error: %v", err)
	}
	if err := prometheus.Register(commitOperationMetrics); err != nil {
		logx.Errorf("prometheus.Register commitOperationMetrics error: %v", err)
	}
	if err := prometheus.Register(verifyLatestHandledMetric); err != nil {
		logx.Errorf("prometheus.Register verifyLatestHandledMetric error: %v", err)
	}
	if err := prometheus.Register(verifyOperationMetrics); err != nil {
		logx.Errorf("prometheus.Register verifyOperationMetrics error: %v", err)
	}
	if err := prometheus.Register(sendPendingMetrics); err != nil {
		logx.Errorf("prometheus.Register sendPendingMetrics error: %v", err)
	}
	if err := prometheus.Register(sendOperationMetrics); err != nil {
		logx.Errorf("prometheus.Register sendOperationMetrics error: %v", err)
	}

	db, err := gorm.Open(postgres.Open(c.Postgres.DataSource))
	if err != nil {
		logx.Errorf("gorm connect db error, err = %v", err)
	}
	s := &Sender{
		config:               c,
		db:                   db,
		blockModel:           block.NewBlockModel(db),
		compressedBlockModel: compressedblock.NewCompressedBlockModel(db),
		l1RollupTxModel:      l1rolluptx.NewL1RollupTxModel(db),
		sysConfigModel:       sysconfig.NewSysConfigModel(db),
		proofModel:           proof.NewProofModel(db),
	}

	l1RPCEndpoint, err := s.sysConfigModel.GetSysConfigByName(c.ChainConfig.NetworkRPCSysConfigName)
	if err != nil {
		logx.Severef("fatal error, cannot fetch l1RPCEndpoint from sysconfig, err: %v, SysConfigName: %s",
			err, c.ChainConfig.NetworkRPCSysConfigName)
		panic(err)
	}
	rollupAddress, err := s.sysConfigModel.GetSysConfigByName(types.ZkBNBContract)
	if err != nil {
		logx.Severef("fatal error, cannot fetch rollupAddress from sysconfig, err: %v, SysConfigName: %s",
			err, types.ZkBNBContract)
		panic(err)
	}

	s.cli, err = rpc.NewClient(l1RPCEndpoint.Value)
	if err != nil {
		panic(err)
	}
	chainId, err := s.cli.ChainID(context.Background())
	if err != nil {
		panic(err)
	}
	s.authCli, err = rpc.NewAuthClient(c.ChainConfig.Sk, chainId)
	if err != nil {
		panic(err)
	}
	s.zkbnbInstance, err = zkbnb.LoadZkBNBInstance(s.cli, rollupAddress.Value)
	if err != nil {
		panic(err)
	}
	return s
}

func (s *Sender) CommitBlocks() (err error) {
	startTime := time.Now()
	var (
		cli           = s.cli
		authCli       = s.authCli
		zkbnbInstance = s.zkbnbInstance
	)
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
	commitLatestHandledMetric.Set(float64(start))
	// commit new blocks
	blocks, err := s.compressedBlockModel.GetCompressedBlocksBetween(start,
		start+int64(s.config.ChainConfig.MaxBlockCount))
	if err != nil && err != types.DbErrNotFound {
		return fmt.Errorf("failed to get compress block err: %v", err)
	}
	if len(blocks) == 0 {
		return nil
	}
	pendingCommitBlocks, err := ConvertBlocksForCommitToCommitBlockInfos(blocks)
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
		gasPrice, err = s.cli.SuggestGasPrice(context.Background())
		if err != nil {
			logx.Errorf("failed to fetch gas price: %v", err)
			return err
		}
	}

	// commit blocks on-chain
	txHash, err := zkbnb.CommitBlocks(
		cli, authCli,
		zkbnbInstance,
		lastStoredBlockInfo,
		pendingCommitBlocks,
		gasPrice,
		s.config.ChainConfig.GasLimit)
	if err != nil {
		return fmt.Errorf("failed to send commit tx, errL %v:%s", err, txHash)
	}
	newRollupTx := &l1rolluptx.L1RollupTx{
		L1TxHash:      txHash,
		TxStatus:      l1rolluptx.StatusPending,
		TxType:        l1rolluptx.TxTypeCommit,
		L2BlockHeight: int64(pendingCommitBlocks[len(pendingCommitBlocks)-1].BlockNumber),
	}
	err = s.l1RollupTxModel.CreateL1RollupTx(newRollupTx)
	if err != nil {
		return fmt.Errorf("failed to create tx in database, err: %v", err)
	}
	logx.Infof("new blocks have been committed(height): %v:%s", newRollupTx.L2BlockHeight, newRollupTx.L1TxHash)
	commitOperationMetrics.Set(float64(time.Since(startTime).Milliseconds()))
	return nil
}

func (s *Sender) UpdateSentTxs() (err error) {
	startTime := time.Now()
	pendingTxs, err := s.l1RollupTxModel.GetL1RollupTxsByStatus(l1rolluptx.StatusPending)
	if err != nil {
		if err == types.DbErrNotFound {
			return nil
		}
		return fmt.Errorf("failed to get pending txs, err: %v", err)
	}
	sendPendingMetrics.Set(float64(len(pendingTxs)))
	latestL1Height, err := s.cli.GetHeight()
	if err != nil {
		return fmt.Errorf("failed to get l1 block height, err: %v", err)
	}

	var (
		pendingUpdateRxs         []*l1rolluptx.L1RollupTx
		pendingUpdateProofStatus = make(map[int64]int)
	)
	for _, pendingTx := range pendingTxs {
		txHash := pendingTx.L1TxHash
		receipt, err := s.cli.GetTransactionReceipt(txHash)
		if err != nil {
			logx.Errorf("query transaction receipt %s failed, err: %v", txHash, err)
			if time.Now().After(pendingTx.UpdatedAt.Add(time.Duration(s.config.ChainConfig.MaxWaitingTime) * time.Second)) {
				// No need to check the response, do best effort.
				logx.Infof("delete timeout l1 rollup tx, tx_hash=%s", pendingTx.L1TxHash)
				//nolint:errcheck
				s.l1RollupTxModel.DeleteL1RollupTx(pendingTx)
			}
			continue
		}
		if receipt.Status == 0 {
			// Should direct mark tx deleted
			logx.Infof("delete timeout l1 rollup tx, tx_hash=%s", pendingTx.L1TxHash)
			//nolint:errcheck
			s.l1RollupTxModel.DeleteL1RollupTx(pendingTx)
			// It is critical to have any failed transactions
			panic(fmt.Sprintf("unexpected failed tx: %v", txHash))
		}

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
		}
	}

	//update db
	err = s.db.Transaction(func(tx *gorm.DB) error {
		//update l1 rollup txs
		err := s.l1RollupTxModel.UpdateL1RollupTxsInTransact(tx, pendingUpdateRxs)
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
	sendOperationMetrics.Set(float64(time.Since(startTime).Milliseconds()))
	return nil
}

func (s *Sender) VerifyAndExecuteBlocks() (err error) {
	startTime := time.Now()
	var (
		cli           = s.cli
		authCli       = s.authCli
		zkbnbInstance = s.zkbnbInstance
	)
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
	verifyLatestHandledMetric.Set(float64(start))
	blocks, err := s.blockModel.GetCommittedBlocksBetween(start,
		start+int64(s.config.ChainConfig.MaxBlockCount))
	if err != nil && err != types.DbErrNotFound {
		return fmt.Errorf("unable to get blocks to prove, err: %v", err)
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
		return fmt.Errorf("unable to get proofs, err: %v", err)
	}
	if len(blockProofs) != len(blocks) {
		return errors.New("related proofs not ready")
	}
	// add sanity check
	for i := range blockProofs {
		if blockProofs[i].BlockNumber != blocks[i].BlockHeight {
			return errors.New("proof number not match")
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
		gasPrice, err = s.cli.SuggestGasPrice(context.Background())
		if err != nil {
			logx.Errorf("failed to fetch gas price: %v", err)
			return err
		}
	}

	// Verify blocks on-chain
	txHash, err := zkbnb.VerifyAndExecuteBlocks(cli, authCli, zkbnbInstance,
		pendingVerifyAndExecuteBlocks, proofs, gasPrice, s.config.ChainConfig.GasLimit)
	if err != nil {
		return fmt.Errorf("failed to send verify tx: %v:%s", err, txHash)
	}

	newRollupTx := &l1rolluptx.L1RollupTx{
		L1TxHash:      txHash,
		TxStatus:      l1rolluptx.StatusPending,
		TxType:        l1rolluptx.TxTypeVerifyAndExecute,
		L2BlockHeight: int64(pendingVerifyAndExecuteBlocks[len(pendingVerifyAndExecuteBlocks)-1].BlockHeader.BlockNumber),
	}
	err = s.l1RollupTxModel.CreateL1RollupTx(newRollupTx)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("failed to create rollup tx in db %v", err))
	}
	logx.Infof("new blocks have been verified and executed(height): %d:%s", newRollupTx.L2BlockHeight, newRollupTx.L1TxHash)
	verifyOperationMetrics.Set(float64(time.Since(startTime).Milliseconds()))
	return nil
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
