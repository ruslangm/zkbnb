package executor

import (
	"github.com/bnb-chain/zkbnb-crypto/wasm/txtypes"
	"math/big"

	sdb "github.com/bnb-chain/zkbnb/core/statedb"
	"github.com/bnb-chain/zkbnb/dao/block"
	"github.com/bnb-chain/zkbnb/dao/tx"
	"github.com/bnb-chain/zkbnb/types"
)

type IBlockchain interface {
	VerifyExpiredAt(expiredAt int64) error
	VerifyNonce(accountIndex int64, nonce int64) error
	VerifyGas(gasAccountIndex, gasFeeAssetId int64, txType int, gasFeeAmount *big.Int, skipGasAmtChk bool) error
	StateDB() *sdb.StateDB
	DB() *sdb.ChainDB
	CurrentBlock() *block.Block
}

type TxExecutor interface {
	Prepare() error
	VerifyInputs(skipGasAmtChk, skipSigChk bool) error
	ApplyTransaction() error
	GeneratePubData() error
	GetExecutedTx(fromApi bool) (*tx.Tx, error)
	GenerateTxDetails() ([]*tx.TxDetail, error)
	GetTxInfo() txtypes.TxInfo
	Finalize() error
}

func NewTxExecutor(bc IBlockchain, tx *tx.Tx) (TxExecutor, error) {
	switch tx.TxType {
	case types.TxTypeChangePubKey:
		return NewChangePubKeyExecutor(bc, tx)
	case types.TxTypeDeposit:
		return NewDepositExecutor(bc, tx)
	case types.TxTypeDepositNft:
		return NewDepositNftExecutor(bc, tx)
	case types.TxTypeTransfer:
		return NewTransferExecutor(bc, tx)
	case types.TxTypeWithdraw:
		return NewWithdrawExecutor(bc, tx)
	case types.TxTypeCreateCollection:
		return NewCreateCollectionExecutor(bc, tx)
	case types.TxTypeMintNft:
		return NewMintNftExecutor(bc, tx)
	case types.TxTypeTransferNft:
		return NewTransferNftExecutor(bc, tx)
	case types.TxTypeAtomicMatch:
		return NewAtomicMatchExecutor(bc, tx)
	case types.TxTypeCancelOffer:
		return NewCancelOfferExecutor(bc, tx)
	case types.TxTypeWithdrawNft:
		return NewWithdrawNftExecutor(bc, tx)
	case types.TxTypeFullExit:
		return NewFullExitExecutor(bc, tx)
	case types.TxTypeFullExitNft:
		return NewFullExitNftExecutor(bc, tx)
	}

	return nil, types.AppErrUnsupportedTxType
}

func NewTxExecutorForDesert(bc IBlockchain, txInfo txtypes.TxInfo) (TxExecutor, error) {
	switch txInfo.GetTxType() {
	case types.TxTypeChangePubKey:
		return NewChangePubKeyExecutorForDesert(bc, txInfo)
	case types.TxTypeDeposit:
		return NewDepositExecutorForDesert(bc, txInfo)
	case types.TxTypeDepositNft:
		return NewDepositNftExecutorForDesert(bc, txInfo)
	case types.TxTypeTransfer:
		return NewTransferExecutorForDesert(bc, txInfo)
	case types.TxTypeWithdraw:
		return NewWithdrawExecutorForDesert(bc, txInfo)
	case types.TxTypeCreateCollection:
		return NewCreateCollectionExecutorForDesert(bc, txInfo)
	case types.TxTypeMintNft:
		return NewMintNftExecutorForDesert(bc, txInfo)
	case types.TxTypeTransferNft:
		return NewTransferNftExecutorForDesert(bc, txInfo)
	case types.TxTypeAtomicMatch:
		return NewAtomicMatchExecutorForDesert(bc, txInfo)
	case types.TxTypeCancelOffer:
		return NewCancelOfferExecutorForDesert(bc, txInfo)
	case types.TxTypeWithdrawNft:
		return NewWithdrawNftExecutorForDesert(bc, txInfo)
	case types.TxTypeFullExit:
		return NewFullExitExecutorForDesert(bc, txInfo)
	case types.TxTypeFullExitNft:
		return NewFullExitNftExecutorForDesert(bc, txInfo)
	}

	return nil, types.AppErrUnsupportedTxType
}
