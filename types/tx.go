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
 *
 */

package types

import (
	"encoding/json"
	"github.com/bnb-chain/zkbnb-crypto/wasm/txtypes"
)

const (
	TxTypeEmpty = iota
	TxTypeRegisterZns
	TxTypeDeposit
	TxTypeDepositNft
	TxTypeTransfer
	TxTypeWithdraw
	TxTypeCreateCollection
	TxTypeMintNft
	TxTypeTransferNft
	TxTypeAtomicMatch
	TxTypeCancelOffer
	TxTypeWithdrawNft
	TxTypeFullExit
	TxTypeFullExitNft
	TxTypeOffer
)

func IsL2Tx(txType int64) bool {
	if txType == TxTypeTransfer ||
		txType == TxTypeWithdraw ||
		txType == TxTypeCreateCollection ||
		txType == TxTypeMintNft ||
		txType == TxTypeTransferNft ||
		txType == TxTypeAtomicMatch ||
		txType == TxTypeCancelOffer ||
		txType == TxTypeWithdrawNft {
		return true
	}
	return false
}

func GetL2TxTypes() []int64 {
	return []int64{TxTypeTransfer,
		TxTypeWithdraw,
		TxTypeCreateCollection,
		TxTypeMintNft,
		TxTypeTransferNft,
		TxTypeAtomicMatch,
		TxTypeCancelOffer,
		TxTypeWithdrawNft}
}

func IsPriorityOperationTx(txType int64) bool {
	if txType == TxTypeRegisterZns ||
		txType == TxTypeDeposit ||
		txType == TxTypeDepositNft ||
		txType == TxTypeFullExit ||
		txType == TxTypeFullExitNft {
		return true
	}
	return false
}

func GetL1TxTypes() []int64 {
	return []int64{TxTypeRegisterZns,
		TxTypeDeposit,
		TxTypeDepositNft,
		TxTypeFullExit,
		TxTypeFullExitNft}
}

const (
	TxTypeBytesSize          = 1
	AddressBytesSize         = 20
	AccountIndexBytesSize    = 4
	AccountNameBytesSize     = 20
	AccountNameHashBytesSize = 32
	PubkeyBytesSize          = 32
	AssetIdBytesSize         = 2
	StateAmountBytesSize     = 16
	NftIndexBytesSize        = 5
	NftTokenIdBytesSize      = 32
	NftContentHashBytesSize  = 32
	FeeRateBytesSize         = 2
	CollectionIdBytesSize    = 2

	RegisterZnsPubDataSize = TxTypeBytesSize + AccountIndexBytesSize + AccountNameBytesSize +
		AccountNameHashBytesSize + PubkeyBytesSize + PubkeyBytesSize
	DepositPubDataSize = TxTypeBytesSize + AccountIndexBytesSize +
		AccountNameHashBytesSize + AssetIdBytesSize + StateAmountBytesSize
	DepositNftPubDataSize = TxTypeBytesSize + AccountIndexBytesSize + NftIndexBytesSize +
		AccountIndexBytesSize + FeeRateBytesSize + NftContentHashBytesSize +
		AccountNameHashBytesSize + CollectionIdBytesSize
	FullExitPubDataSize = TxTypeBytesSize + AccountIndexBytesSize +
		AccountNameHashBytesSize + AssetIdBytesSize + StateAmountBytesSize
	FullExitNftPubDataSize = TxTypeBytesSize + AccountIndexBytesSize + AccountIndexBytesSize + FeeRateBytesSize +
		NftIndexBytesSize + CollectionIdBytesSize +
		AccountNameHashBytesSize + AccountNameHashBytesSize +
		NftContentHashBytesSize
)

const (
	TypeAccountIndex = iota
	TypeAssetId
	TypeAccountName
	TypeAccountNameOmitSpace
	TypeAccountPk
	TypeLimit
	TypeOffset
	TypeHash
	TypeBlockHeight
	TypeTxType
	TypeChainId
	TypeAssetAmount
	TypeBoolean
	TypeGasFee
)

const (
	AddressSize       = 20
	EmptyStringKeccak = "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"
)

type UpdateNftReq struct {
	NftIndex          int64
	MutableAttributes string
	AccountIndex      int64
	Nonce             int64
}

func ParseRegisterZnsTxInfo(txInfoStr string) (txInfo *txtypes.RegisterZnsTxInfo, err error) {
	err = json.Unmarshal([]byte(txInfoStr), &txInfo)
	if err != nil {
		return nil, err
	}
	return txInfo, nil
}

func ParseDepositTxInfo(txInfoStr string) (txInfo *txtypes.DepositTxInfo, err error) {
	err = json.Unmarshal([]byte(txInfoStr), &txInfo)
	if err != nil {
		return nil, err
	}
	return txInfo, nil
}

func ParseDepositNftTxInfo(txInfoStr string) (txInfo *txtypes.DepositNftTxInfo, err error) {
	err = json.Unmarshal([]byte(txInfoStr), &txInfo)
	if err != nil {
		return nil, err
	}
	return txInfo, nil
}

func ParseFullExitTxInfo(txInfoStr string) (txInfo *txtypes.FullExitTxInfo, err error) {
	err = json.Unmarshal([]byte(txInfoStr), &txInfo)
	if err != nil {
		return nil, err
	}
	return txInfo, nil
}

func ParseFullExitNftTxInfo(txInfoStr string) (txInfo *txtypes.FullExitNftTxInfo, err error) {
	err = json.Unmarshal([]byte(txInfoStr), &txInfo)
	if err != nil {
		return nil, err
	}
	return txInfo, nil
}

func ParseCreateCollectionTxInfo(txInfoStr string) (txInfo *txtypes.CreateCollectionTxInfo, err error) {
	err = json.Unmarshal([]byte(txInfoStr), &txInfo)
	if err != nil {
		return nil, err
	}
	return txInfo, nil
}

func ParseTransferTxInfo(txInfoStr string) (txInfo *txtypes.TransferTxInfo, err error) {
	err = json.Unmarshal([]byte(txInfoStr), &txInfo)
	if err != nil {
		return nil, err
	}
	return txInfo, nil
}

func ParseMintNftTxInfo(txInfoStr string) (txInfo *txtypes.MintNftTxInfo, err error) {
	err = json.Unmarshal([]byte(txInfoStr), &txInfo)
	if err != nil {
		return nil, err
	}
	return txInfo, nil
}

func ParseTransferNftTxInfo(txInfoStr string) (txInfo *txtypes.TransferNftTxInfo, err error) {
	err = json.Unmarshal([]byte(txInfoStr), &txInfo)
	if err != nil {
		return nil, err
	}
	return txInfo, nil
}

func ParseAtomicMatchTxInfo(txInfoStr string) (txInfo *txtypes.AtomicMatchTxInfo, err error) {
	err = json.Unmarshal([]byte(txInfoStr), &txInfo)
	if err != nil {
		return nil, err
	}
	return txInfo, nil
}

func ParseCancelOfferTxInfo(txInfoStr string) (txInfo *txtypes.CancelOfferTxInfo, err error) {
	err = json.Unmarshal([]byte(txInfoStr), &txInfo)
	if err != nil {
		return nil, err
	}
	return txInfo, nil
}

func ParseWithdrawTxInfo(txInfoStr string) (txInfo *txtypes.WithdrawTxInfo, err error) {
	err = json.Unmarshal([]byte(txInfoStr), &txInfo)
	if err != nil {
		return nil, err
	}
	return txInfo, nil
}

func ParseWithdrawNftTxInfo(txInfoStr string) (txInfo *txtypes.WithdrawNftTxInfo, err error) {
	err = json.Unmarshal([]byte(txInfoStr), &txInfo)
	if err != nil {
		return nil, err
	}
	return txInfo, nil
}

func ParseUpdateNftTxInfo(txInfoStr string) (txInfo *UpdateNftReq, err error) {
	err = json.Unmarshal([]byte(txInfoStr), &txInfo)
	if err != nil {
		return nil, err
	}
	return txInfo, nil
}
