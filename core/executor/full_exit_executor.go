package executor

import (
	"bytes"
	"encoding/json"
	"github.com/bnb-chain/zkbnb/common/chain"
	"github.com/bnb-chain/zkbnb/tree"
	"math/big"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/bnb-chain/zkbnb-crypto/ffmath"
	"github.com/bnb-chain/zkbnb-crypto/wasm/txtypes"
	common2 "github.com/bnb-chain/zkbnb/common"
	"github.com/bnb-chain/zkbnb/dao/tx"
	"github.com/bnb-chain/zkbnb/types"
)

type FullExitExecutor struct {
	BaseExecutor

	TxInfo          *txtypes.FullExitTxInfo
	AccountNotExist bool
}

func NewFullExitExecutor(bc IBlockchain, tx *tx.Tx) (TxExecutor, error) {
	txInfo, err := types.ParseFullExitTxInfo(tx.TxInfo)
	if err != nil {
		logx.Errorf("parse full exit tx failed: %s", err.Error())
		return nil, types.AppErrInvalidTxInfo
	}

	return &FullExitExecutor{
		BaseExecutor: NewBaseExecutor(bc, tx, txInfo, false),
		TxInfo:       txInfo,
	}, nil
}

func NewFullExitExecutorForDesert(bc IBlockchain, txInfo txtypes.TxInfo) (TxExecutor, error) {
	return &FullExitExecutor{
		BaseExecutor: NewBaseExecutor(bc, nil, txInfo, true),
		TxInfo:       txInfo.(*txtypes.FullExitTxInfo),
	}, nil
}

func (e *FullExitExecutor) Prepare() error {
	bc := e.bc
	txInfo := e.TxInfo
	txInfo.AssetAmount = new(big.Int).SetInt64(0)
	formatAccountByIndex, err := bc.StateDB().GetFormatAccount(txInfo.AccountIndex)
	if err != nil && err != types.AppErrAccountNotFound {
		return err
	}
	if err == types.AppErrAccountNotFound {
		e.AccountNotExist = true
		return nil
	}
	if formatAccountByIndex.L1Address == txInfo.L1Address {
		// Set the right asset amount.
		if formatAccountByIndex.AssetInfo == nil || formatAccountByIndex.AssetInfo[txInfo.AssetId] == nil {
			txInfo.AssetAmount = new(big.Int).SetInt64(0)
		} else {
			txInfo.AssetAmount = formatAccountByIndex.AssetInfo[txInfo.AssetId].Balance
		}
	}

	// Mark the tree states that would be affected in this executor.
	e.MarkAccountAssetsDirty(txInfo.AccountIndex, []int64{txInfo.AssetId})
	err = e.BaseExecutor.Prepare()
	if err != nil {
		return err
	}
	return nil
}

func (e *FullExitExecutor) VerifyInputs(skipGasAmtChk, skipSigChk bool) error {
	return nil
}

func (e *FullExitExecutor) ApplyTransaction() error {
	if e.AccountNotExist {
		return nil
	}
	bc := e.bc
	txInfo := e.TxInfo

	exitAccount, err := bc.StateDB().GetFormatAccount(txInfo.AccountIndex)
	if err != nil {
		return err
	}
	exitAccount.AssetInfo[txInfo.AssetId].Balance = ffmath.Sub(exitAccount.AssetInfo[txInfo.AssetId].Balance, txInfo.AssetAmount)

	if txInfo.AssetAmount.Cmp(types.ZeroBigInt) != 0 {
		stateCache := e.bc.StateDB()
		stateCache.SetPendingAccount(txInfo.AccountIndex, exitAccount)
	}
	return e.BaseExecutor.ApplyTransaction()
}

func (e *FullExitExecutor) GeneratePubData() error {
	txInfo := e.TxInfo

	var buf bytes.Buffer
	buf.WriteByte(uint8(types.TxTypeFullExit))
	buf.Write(common2.Uint32ToBytes(uint32(txInfo.AccountIndex)))
	buf.Write(common2.Uint16ToBytes(uint16(txInfo.AssetId)))
	buf.Write(common2.Uint128ToBytes(txInfo.AssetAmount))
	buf.Write(common2.AddressStrToBytes(txInfo.L1Address))

	pubData := common2.SuffixPaddingBuToPubdataSize(buf.Bytes())

	stateCache := e.bc.StateDB()
	stateCache.PriorityOperations++
	stateCache.PubDataOffset = append(stateCache.PubDataOffset, uint32(len(stateCache.PubData)))
	stateCache.PendingOnChainOperationsPubData = append(stateCache.PendingOnChainOperationsPubData, pubData)
	stateCache.PendingOnChainOperationsHash = common2.ConcatKeccakHash(stateCache.PendingOnChainOperationsHash, pubData)
	stateCache.PubData = append(stateCache.PubData, pubData...)
	return nil
}

func (e *FullExitExecutor) GetExecutedTx(fromApi bool) (*tx.Tx, error) {
	txInfoBytes, err := json.Marshal(e.TxInfo)
	if err != nil {
		logx.Errorf("unable to marshal tx, err: %s", err.Error())
		return nil, types.AppErrMarshalTxFailed
	}

	e.tx.TxInfo = string(txInfoBytes)
	e.tx.AssetId = e.TxInfo.AssetId
	e.tx.TxAmount = e.TxInfo.AssetAmount.String()
	return e.BaseExecutor.GetExecutedTx(fromApi)
}

func (e *FullExitExecutor) GenerateTxDetails() ([]*tx.TxDetail, error) {
	txInfo := e.TxInfo
	var exitAccount *types.AccountInfo
	var err error
	if e.AccountNotExist {
		exitAccount, err = chain.EmptyAccountFormat(txInfo.AccountIndex, []int64{txInfo.AssetId}, types.EmptyL1Address, tree.NilAccountAssetRoot)
		if err != nil {
			return nil, err
		}
	} else {
		exitAccount, err = e.bc.StateDB().GetFormatAccount(txInfo.AccountIndex)
		if err != nil {
			return nil, err
		}
	}

	baseBalance := exitAccount.AssetInfo[txInfo.AssetId]
	deltaBalance := &types.AccountAsset{
		AssetId:                  txInfo.AssetId,
		Balance:                  ffmath.Neg(txInfo.AssetAmount),
		OfferCanceledOrFinalized: big.NewInt(0),
	}
	txDetail := &tx.TxDetail{
		AssetId:         txInfo.AssetId,
		AssetType:       types.FungibleAssetType,
		AccountIndex:    txInfo.AccountIndex,
		L1Address:       exitAccount.L1Address,
		Balance:         baseBalance.String(),
		BalanceDelta:    deltaBalance.String(),
		Order:           0,
		AccountOrder:    0,
		Nonce:           exitAccount.Nonce,
		CollectionNonce: exitAccount.CollectionNonce,
		PublicKey:       exitAccount.PublicKey,
	}
	return []*tx.TxDetail{txDetail}, nil
}
