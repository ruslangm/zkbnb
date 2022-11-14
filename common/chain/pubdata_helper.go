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

package chain

import (
	"errors"

	"github.com/consensys/gnark-crypto/ecc/bn254/twistededwards/eddsa"
	"github.com/ethereum/go-ethereum/common"

	"github.com/bnb-chain/zkbnb-crypto/wasm/txtypes"
	common2 "github.com/bnb-chain/zkbnb/common"
	"github.com/bnb-chain/zkbnb/types"
)

func ParseRegisterZnsPubData(pubData []byte) (tx *txtypes.RegisterZnsTxInfo, err error) {
	/*
		struct RegisterZNS {
			uint8 txType;
			bytes20 accountName;
			bytes32 accountNameHash;
			bytes32 pubKeyX;
			bytes32 pubKeyY;
		}
	*/
	if len(pubData) != types.RegisterZnsPubDataSize {
		return nil, errors.New("[ParseRegisterZnsPubData] invalid size")
	}
	offset := 0
	offset, txType := common2.ReadUint8(pubData, offset)
	offset, accountIndex := common2.ReadUint32(pubData, offset)
	offset, accountName := common2.ReadBytes20(pubData, offset)
	offset, accountNameHash := common2.ReadBytes32(pubData, offset)
	offset, pubKeyX := common2.ReadBytes32(pubData, offset)
	_, pubKeyY := common2.ReadBytes32(pubData, offset)
	pk := new(eddsa.PublicKey)
	pk.A.X.SetBytes(pubKeyX)
	pk.A.Y.SetBytes(pubKeyY)
	tx = &txtypes.RegisterZnsTxInfo{
		TxType:          txType,
		AccountIndex:    int64(accountIndex),
		AccountName:     common2.CleanAccountName(common2.SerializeAccountName(accountName)),
		AccountNameHash: accountNameHash,
		PubKey:          common.Bytes2Hex(pk.Bytes()),
	}
	return tx, nil
}

func ParseDepositPubData(pubData []byte) (tx *txtypes.DepositTxInfo, err error) {
	/*
		struct Deposit {
			uint8 txType;
			uint32 accountIndex;
			bytes32 accountNameHash;
			uint16 assetId;
			uint128 amount;
		}
	*/
	if len(pubData) != types.DepositPubDataSize {
		return nil, errors.New("[ParseDepositPubData] invalid size")
	}
	offset := 0
	offset, txType := common2.ReadUint8(pubData, offset)
	offset, accountIndex := common2.ReadUint32(pubData, offset)
	offset, accountNameHash := common2.ReadBytes32(pubData, offset)
	offset, assetId := common2.ReadUint16(pubData, offset)
	_, amount := common2.ReadUint128(pubData, offset)
	tx = &txtypes.DepositTxInfo{
		TxType:          txType,
		AccountIndex:    int64(accountIndex),
		AccountNameHash: accountNameHash,
		AssetId:         int64(assetId),
		AssetAmount:     amount,
	}
	return tx, nil
}

func ParseDepositNftPubData(pubData []byte) (tx *txtypes.DepositNftTxInfo, err error) {
	if len(pubData) != types.DepositNftPubDataSize {
		return nil, errors.New("[ParseDepositNftPubData] invalid size")
	}
	offset := 0
	offset, txType := common2.ReadUint8(pubData, offset)
	offset, accountIndex := common2.ReadUint32(pubData, offset)
	offset, nftIndex := common2.ReadUint40(pubData, offset)
	offset, creatorAccountIndex := common2.ReadUint32(pubData, offset)
	offset, creatorTreasuryRate := common2.ReadUint16(pubData, offset)
	offset, nftContentHash := common2.ReadBytes32(pubData, offset)
	offset, accountNameHash := common2.ReadBytes32(pubData, offset)
	_, collectionId := common2.ReadUint16(pubData, offset)
	tx = &txtypes.DepositNftTxInfo{
		TxType:              txType,
		AccountIndex:        int64(accountIndex),
		NftIndex:            nftIndex,
		CreatorAccountIndex: int64(creatorAccountIndex),
		CreatorTreasuryRate: int64(creatorTreasuryRate),
		NftContentHash:      nftContentHash,
		AccountNameHash:     accountNameHash,
		CollectionId:        int64(collectionId),
	}
	return tx, nil
}

func ParseFullExitPubData(pubData []byte) (tx *txtypes.FullExitTxInfo, err error) {
	if len(pubData) != types.FullExitPubDataSize {
		return nil, errors.New("[ParseFullExitPubData] invalid size")
	}
	offset := 0
	offset, txType := common2.ReadUint8(pubData, offset)
	offset, accountIndex := common2.ReadUint32(pubData, offset)
	offset, assetId := common2.ReadUint16(pubData, offset)
	offset, assetAmount := common2.ReadUint128(pubData, offset)
	_, accountNameHash := common2.ReadBytes32(pubData, offset)
	tx = &txtypes.FullExitTxInfo{
		TxType:          txType,
		AccountIndex:    int64(accountIndex),
		AccountNameHash: accountNameHash,
		AssetId:         int64(assetId),
		AssetAmount:     assetAmount,
	}
	return tx, nil
}

func ParseFullExitNftPubData(pubData []byte) (tx *txtypes.FullExitNftTxInfo, err error) {
	if len(pubData) != types.FullExitNftPubDataSize {
		return nil, errors.New("[ParseFullExitNftPubData] invalid size")
	}
	offset := 0
	offset, txType := common2.ReadUint8(pubData, offset)
	offset, accountIndex := common2.ReadUint32(pubData, offset)
	offset, creatorAccountIndex := common2.ReadUint32(pubData, offset)
	offset, creatorTreasuryRate := common2.ReadUint16(pubData, offset)
	offset, nftIndex := common2.ReadUint40(pubData, offset)
	offset, collectionId := common2.ReadUint16(pubData, offset)
	offset, accountNameHash := common2.ReadBytes32(pubData, offset)
	offset, creatorAccountNameHash := common2.ReadBytes32(pubData, offset)
	_, nftContentHash := common2.ReadBytes32(pubData, offset)
	tx = &txtypes.FullExitNftTxInfo{
		TxType:                 txType,
		AccountIndex:           int64(accountIndex),
		CreatorAccountIndex:    int64(creatorAccountIndex),
		CreatorTreasuryRate:    int64(creatorTreasuryRate),
		NftIndex:               nftIndex,
		CollectionId:           int64(collectionId),
		AccountNameHash:        accountNameHash,
		CreatorAccountNameHash: creatorAccountNameHash,
		NftContentHash:         nftContentHash,
	}
	return tx, nil
}
