package executor

import (
	"github.com/bnb-chain/zkbnb/common/metrics"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/zeromicro/go-zero/core/logx"
	"time"

	"github.com/bnb-chain/zkbnb-crypto/wasm/txtypes"
	"github.com/bnb-chain/zkbnb/dao/tx"
	"github.com/bnb-chain/zkbnb/types"
)

const (
	OfferPerAsset = 128
	TenThousand   = 10000
)

type BaseExecutor struct {
	bc      IBlockchain
	tx      *tx.Tx
	iTxInfo txtypes.TxInfo

	// Affected states.
	dirtyAccountsAndAssetsMap map[int64]map[int64]bool
	dirtyNftMap               map[int64]bool
}

func NewBaseExecutor(bc IBlockchain, tx *tx.Tx, txInfo txtypes.TxInfo) BaseExecutor {
	return BaseExecutor{
		bc:      bc,
		tx:      tx,
		iTxInfo: txInfo,

		dirtyAccountsAndAssetsMap: make(map[int64]map[int64]bool, 0),
		dirtyNftMap:               make(map[int64]bool, 0),
	}
}

func (e *BaseExecutor) Prepare() error {
	// Assign tx related fields for layer2 transaction from the API.
	from := e.iTxInfo.GetFromAccountIndex()
	if from != types.NilAccountIndex && e.tx.TxHash == types.EmptyTxHash {
		// Compute tx hash for layer2 transactions.
		hash, err := e.iTxInfo.Hash(mimc.NewMiMC())
		if err != nil {
			return err
		}
		e.tx.TxHash = common.Bytes2Hex(hash)
		e.tx.AccountIndex = e.iTxInfo.GetFromAccountIndex()
		e.tx.Nonce = e.iTxInfo.GetNonce()
		e.tx.ExpiredAt = e.iTxInfo.GetExpiredAt()
	}

	err := e.bc.StateDB().PrepareAccountsAndAssets(e.dirtyAccountsAndAssetsMap)
	if err != nil {
		logx.Errorf("prepare accounts and assets failed: %s", err.Error())
		return err
	}
	return nil
}

func (e *BaseExecutor) VerifyInputs(skipGasAmtChk, skipSigChk bool) error {
	txInfo := e.iTxInfo

	err := txInfo.Validate()
	if err != nil {
		return err
	}
	err = e.bc.VerifyExpiredAt(txInfo.GetExpiredAt())
	if err != nil {
		return err
	}

	from := txInfo.GetFromAccountIndex()
	if from != types.NilAccountIndex {
		err = e.bc.VerifyNonce(from, txInfo.GetNonce())
		if err != nil {
			return err
		}

		gasAccountIndex, gasFeeAssetId, gasFeeAmount := txInfo.GetGas()
		var start time.Time
		start = time.Now()
		err = e.bc.VerifyGas(gasAccountIndex, gasFeeAssetId, txInfo.GetTxType(), gasFeeAmount, skipGasAmtChk)
		if metrics.VerifyGasGauge != nil {
			metrics.VerifyGasGauge.Set(float64(time.Since(start).Milliseconds()))
		}

		if err != nil {
			return err
		}

		if !skipSigChk {
			fromAccount, err := e.bc.StateDB().GetFormatAccount(from)
			if err != nil {
				return err
			}
			start = time.Now()
			err = txInfo.VerifySignature(fromAccount.PublicKey)
			if metrics.VerifySignature != nil {
				metrics.VerifySignature.Set(float64(time.Since(start).Milliseconds()))
			}
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (e *BaseExecutor) ApplyTransaction() error {
	e.SyncDirtyToStateCache()
	return nil
}

func (e *BaseExecutor) GeneratePubData() error {
	return nil
}

func (e *BaseExecutor) GetExecutedTx(fromApi bool) (*tx.Tx, error) {
	if fromApi {
		return e.tx, nil
	}
	e.tx.TxIndex = int64(len(e.bc.StateDB().Txs))
	e.tx.BlockHeight = e.bc.CurrentBlock().BlockHeight
	e.tx.TxStatus = tx.StatusExecuted
	e.tx.PoolTxId = e.tx.ID
	e.tx.BlockId = e.bc.CurrentBlock().ID
	return e.tx, nil
}

func (e *BaseExecutor) GenerateTxDetails() ([]*tx.TxDetail, error) {
	return nil, nil
}

func (e *BaseExecutor) MarkAccountAssetsDirty(accountIndex int64, assets []int64) {
	if accountIndex < 0 {
		return
	}

	_, ok := e.dirtyAccountsAndAssetsMap[accountIndex]
	if !ok {
		e.dirtyAccountsAndAssetsMap[accountIndex] = make(map[int64]bool, 0)
	}

	for _, assetIndex := range assets {
		// Should never happen, but protect here.
		if assetIndex < 0 {
			continue
		}
		e.dirtyAccountsAndAssetsMap[accountIndex][assetIndex] = true
	}
}

func (e *BaseExecutor) MarkNftDirty(nftIndex int64) {
	e.dirtyNftMap[nftIndex] = true
}

func (e *BaseExecutor) SyncDirtyToStateCache() {
	for accountIndex, assetsMap := range e.dirtyAccountsAndAssetsMap {
		assets := make([]int64, 0, len(assetsMap))
		for assetIndex := range assetsMap {
			assets = append(assets, assetIndex)
		}
		e.bc.StateDB().MarkAccountAssetsDirty(accountIndex, assets)
	}

	for nftIndex := range e.dirtyNftMap {
		e.bc.StateDB().MarkNftDirty(nftIndex)
	}
}
