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

package prove

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/bnb-chain/zkbnb-crypto/circuit"
	cryptoTypes "github.com/bnb-chain/zkbnb-crypto/circuit/types"
	"github.com/bnb-chain/zkbnb-crypto/ffmath"
	bsmt "github.com/bnb-chain/zkbnb-smt"
	common2 "github.com/bnb-chain/zkbnb/common"
	"github.com/bnb-chain/zkbnb/common/chain"
	"github.com/bnb-chain/zkbnb/dao/account"
	"github.com/bnb-chain/zkbnb/dao/block"
	"github.com/bnb-chain/zkbnb/dao/tx"
	"github.com/bnb-chain/zkbnb/tree"
	"github.com/bnb-chain/zkbnb/types"
)

type WitnessHelper struct {
	treeCtx *tree.Context

	accountModel        account.AccountModel
	accountHistoryModel account.AccountHistoryModel

	gasAccountInfo *types.AccountInfo //gas account cache

	// Trees
	accountTree bsmt.SparseMerkleTree
	assetTrees  *tree.AssetTreeCache
	nftTree     bsmt.SparseMerkleTree
}

func NewWitnessHelper(treeCtx *tree.Context, accountTree, nftTree bsmt.SparseMerkleTree,
	assetTrees *tree.AssetTreeCache, accountModel account.AccountModel, accountHistoryModel account.AccountHistoryModel) *WitnessHelper {
	return &WitnessHelper{
		treeCtx:             treeCtx,
		accountModel:        accountModel,
		accountHistoryModel: accountHistoryModel,
		accountTree:         accountTree,
		assetTrees:          assetTrees,
		nftTree:             nftTree,
	}
}

func (w *WitnessHelper) ConstructTxWitness(oTx *tx.Tx, finalityBlockNr uint64,
) (cryptoTx *TxWitness, err error) {
	switch oTx.TxType {
	case types.TxTypeEmpty:
		return nil, fmt.Errorf("there should be no empty tx")
	default:
		cryptoTx, err = w.constructTxWitness(oTx, finalityBlockNr)
		if err != nil {
			return nil, err
		}
	}
	return cryptoTx, nil
}

func (w *WitnessHelper) constructTxWitness(oTx *tx.Tx, finalityBlockNr uint64) (witness *TxWitness, err error) {
	if oTx == nil || w.accountTree == nil || w.assetTrees == nil || w.nftTree == nil {
		return nil, fmt.Errorf("failed because of nil tx or tree")
	}
	witness, err = w.constructWitnessInfo(oTx, finalityBlockNr)
	if err != nil {
		return nil, err
	}
	witness.TxType = uint8(oTx.TxType)
	witness.Nonce = oTx.Nonce
	switch oTx.TxType {
	case types.TxTypeRegisterZns:
		return w.constructRegisterZnsTxWitness(witness, oTx)
	case types.TxTypeDeposit:
		return w.constructDepositTxWitness(witness, oTx)
	case types.TxTypeDepositNft:
		return w.constructDepositNftTxWitness(witness, oTx)
	case types.TxTypeTransfer:
		return w.constructTransferTxWitness(witness, oTx)
	case types.TxTypeWithdraw:
		return w.constructWithdrawTxWitness(witness, oTx)
	case types.TxTypeCreateCollection:
		return w.constructCreateCollectionTxWitness(witness, oTx)
	case types.TxTypeMintNft:
		return w.constructMintNftTxWitness(witness, oTx)
	case types.TxTypeTransferNft:
		return w.constructTransferNftTxWitness(witness, oTx)
	case types.TxTypeAtomicMatch:
		return w.constructAtomicMatchTxWitness(witness, oTx)
	case types.TxTypeCancelOffer:
		return w.constructCancelOfferTxWitness(witness, oTx)
	case types.TxTypeWithdrawNft:
		return w.constructWithdrawNftTxWitness(witness, oTx)
	case types.TxTypeFullExit:
		return w.constructFullExitTxWitness(witness, oTx)
	case types.TxTypeFullExitNft:
		return w.constructFullExitNftTxWitness(witness, oTx)
	default:
		return nil, fmt.Errorf("tx type error")
	}
}

func (w *WitnessHelper) constructWitnessInfo(
	oTx *tx.Tx,
	finalityBlockNr uint64,
) (
	cryptoTx *TxWitness,
	err error,
) {
	accountKeys, proverAccounts, proverNftInfo, err := w.constructSimpleWitnessInfo(oTx)
	if err != nil {
		return nil, err
	}
	// construct account witness
	accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore, err :=
		w.constructAccountWitness(oTx, finalityBlockNr, accountKeys, proverAccounts)
	if err != nil {
		return nil, err
	}
	// construct nft witness
	nftRootBefore, nftBefore, merkleProofsNftBefore, err :=
		w.constructNftWitness(proverNftInfo)
	if err != nil {
		return nil, err
	}
	stateRootBefore := tree.ComputeStateRootHash(accountRootBefore, nftRootBefore)
	stateRootAfter := tree.ComputeStateRootHash(w.accountTree.Root(), w.nftTree.Root())
	cryptoTx = &TxWitness{
		AccountRootBefore:               accountRootBefore,
		AccountsInfoBefore:              accountsInfoBefore,
		NftRootBefore:                   nftRootBefore,
		NftBefore:                       nftBefore,
		StateRootBefore:                 stateRootBefore,
		MerkleProofsAccountAssetsBefore: merkleProofsAccountAssetsBefore,
		MerkleProofsAccountBefore:       merkleProofsAccountBefore,
		MerkleProofsNftBefore:           merkleProofsNftBefore,
		StateRootAfter:                  stateRootAfter,
	}
	return cryptoTx, nil
}

func (w *WitnessHelper) constructAccountWitness(
	oTx *tx.Tx,
	finalityBlockNr uint64,
	accountKeys []int64,
	proverAccounts []*AccountWitnessInfo,
) (
	accountRootBefore []byte,
	// account before info, size is 4
	accountsInfoBefore [NbAccountsPerTx]*cryptoTypes.Account,
	// before account asset merkle proof
	merkleProofsAccountAssetsBefore [NbAccountsPerTx][NbAccountAssetsPerAccount][AssetMerkleLevels][]byte,
	// before account merkle proof
	merkleProofsAccountBefore [NbAccountsPerTx][AccountMerkleLevels][]byte,
	err error,
) {
	accountRootBefore = w.accountTree.Root()
	var (
		accountCount = 0
	)
	for _, accountKey := range accountKeys {
		var (
			cryptoAccount = new(cryptoTypes.Account)
			// get account asset before
			assetCount = 0
		)
		// get account before
		accountMerkleProofs, err := w.accountTree.GetProof(uint64(accountKey))
		if err != nil {
			return accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore, err
		}
		// it means this is a registerZNS tx
		if proverAccounts == nil {
			if accountKey != w.assetTrees.GetNextAccountIndex() {
				return accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore,
					fmt.Errorf("invalid key")
			}
			w.assetTrees.UpdateCache(accountKey, int64(finalityBlockNr))
			cryptoAccount = cryptoTypes.EmptyAccount(accountKey, tree.NilAccountAssetRoot)
			// update account info
			accountInfo, err := w.accountModel.GetConfirmedAccountByIndex(accountKey)
			if err != nil {
				return accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore, err
			}
			proverAccounts = append(proverAccounts, &AccountWitnessInfo{
				AccountInfo: &account.Account{
					AccountIndex:    accountInfo.AccountIndex,
					AccountName:     accountInfo.AccountName,
					PublicKey:       accountInfo.PublicKey,
					AccountNameHash: accountInfo.AccountNameHash,
					L1Address:       accountInfo.L1Address,
					Nonce:           types.EmptyNonce,
					CollectionNonce: types.EmptyCollectionNonce,
					AssetInfo:       types.EmptyAccountAssetInfo,
					AssetRoot:       common.Bytes2Hex(tree.NilAccountAssetRoot),
					Status:          accountInfo.Status,
				},
			})

			// cache gas account
			if accountKey == types.GasAccount {
				w.gasAccountInfo = &types.AccountInfo{
					AccountIndex:    accountInfo.AccountIndex,
					AccountName:     accountInfo.AccountName,
					PublicKey:       accountInfo.PublicKey,
					AccountNameHash: accountInfo.AccountNameHash,
					Nonce:           types.EmptyNonce,
					CollectionNonce: types.EmptyCollectionNonce,
					AssetRoot:       common.Bytes2Hex(tree.NilAccountAssetRoot),
					AssetInfo:       make(map[int64]*types.AccountAsset, 0),
				}
			}
		} else {
			proverAccountInfo := proverAccounts[accountCount]
			pk, err := common2.ParsePubKey(proverAccountInfo.AccountInfo.PublicKey)
			if err != nil {
				return accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore, err
			}
			cryptoAccount = &cryptoTypes.Account{
				AccountIndex:    accountKey,
				AccountNameHash: common.FromHex(proverAccountInfo.AccountInfo.AccountNameHash),
				AccountPk:       pk,
				Nonce:           proverAccountInfo.AccountInfo.Nonce,
				CollectionNonce: proverAccountInfo.AccountInfo.CollectionNonce,
				AssetRoot:       w.assetTrees.Get(accountKey).Root(),
			}
			for i, accountAsset := range proverAccountInfo.AccountAssets {
				assetMerkleProof, err := w.assetTrees.Get(accountKey).GetProof(uint64(accountAsset.AssetId))
				if err != nil {
					return accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore, err
				}
				// set crypto account asset
				cryptoAccount.AssetsInfo[assetCount] = &cryptoTypes.AccountAsset{
					AssetId:                  accountAsset.AssetId,
					Balance:                  accountAsset.Balance,
					OfferCanceledOrFinalized: accountAsset.OfferCanceledOrFinalized,
				}

				// set merkle proof
				merkleProofsAccountAssetsBefore[accountCount][assetCount], err = SetFixedAccountAssetArray(assetMerkleProof)
				if err != nil {
					return accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore, err
				}
				// update asset merkle tree
				nBalance, err := chain.ComputeNewBalance(
					proverAccountInfo.AssetsRelatedTxDetails[i].AssetType,
					proverAccountInfo.AssetsRelatedTxDetails[i].Balance,
					proverAccountInfo.AssetsRelatedTxDetails[i].BalanceDelta,
				)
				if err != nil {
					return accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore, err
				}
				nAsset, err := types.ParseAccountAsset(nBalance)
				if err != nil {
					return accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore, err
				}
				nAssetHash, err := tree.ComputeAccountAssetLeafHash(nAsset.Balance.String(), nAsset.OfferCanceledOrFinalized.String())
				if err != nil {
					return accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore, err
				}
				err = w.assetTrees.Get(accountKey).Set(uint64(accountAsset.AssetId), nAssetHash)
				if err != nil {
					return accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore, err
				}

				assetCount++
			}
			if err != nil {
				return accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore, err
			}
		}
		// padding empty account asset
		for assetCount < NbAccountAssetsPerAccount {
			cryptoAccount.AssetsInfo[assetCount] = cryptoTypes.EmptyAccountAsset(LastAccountAssetId)
			assetMerkleProof, err := w.assetTrees.Get(accountKey).GetProof(LastAccountAssetId)
			if err != nil {
				return accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore, err
			}
			merkleProofsAccountAssetsBefore[accountCount][assetCount], err = SetFixedAccountAssetArray(assetMerkleProof)
			if err != nil {
				return accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore, err
			}
			assetCount++
		}
		// set account merkle proof
		merkleProofsAccountBefore[accountCount], err = SetFixedAccountArray(accountMerkleProofs)
		if err != nil {
			return accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore, err
		}
		// update account merkle tree
		nonce := cryptoAccount.Nonce
		collectionNonce := cryptoAccount.CollectionNonce
		if oTx.AccountIndex == accountKey && types.IsL2Tx(oTx.TxType) {
			nonce = oTx.Nonce + 1 // increase nonce if tx is initiated in l2
		}
		if oTx.AccountIndex == accountKey && oTx.TxType == types.TxTypeCreateCollection {
			collectionNonce++
		}
		nAccountHash, err := tree.ComputeAccountLeafHash(
			proverAccounts[accountCount].AccountInfo.AccountNameHash,
			proverAccounts[accountCount].AccountInfo.PublicKey,
			nonce,
			collectionNonce,
			w.assetTrees.Get(accountKey).Root(),
		)
		if err != nil {
			return accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore, err
		}
		err = w.accountTree.Set(uint64(accountKey), nAccountHash)
		if err != nil {
			return accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore, err
		}

		// cache gas account
		if accountKey == types.GasAccount {
			w.gasAccountInfo.Nonce = nonce
			w.gasAccountInfo.CollectionNonce = collectionNonce
			w.gasAccountInfo.AssetRoot = common.Bytes2Hex(w.assetTrees.Get(accountKey).Root())
			for i := 0; i < NbAccountAssetsPerAccount; i++ {
				w.gasAccountInfo.AssetInfo[cryptoAccount.AssetsInfo[i].AssetId] = &types.AccountAsset{
					AssetId:                  cryptoAccount.AssetsInfo[i].AssetId,
					Balance:                  cryptoAccount.AssetsInfo[i].Balance,
					OfferCanceledOrFinalized: cryptoAccount.AssetsInfo[i].OfferCanceledOrFinalized,
				}
			}
		}

		// set account info before
		accountsInfoBefore[accountCount] = cryptoAccount
		// add count
		accountCount++
	}
	if err != nil {
		return accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore, err
	}
	// padding empty account
	emptyAssetTree, err := tree.NewMemAccountAssetTree()
	if err != nil {
		return accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore, err
	}
	for accountCount < NbAccountsPerTx {
		accountsInfoBefore[accountCount] = cryptoTypes.EmptyAccount(LastAccountIndex, tree.NilAccountAssetRoot)
		// get account before
		accountMerkleProofs, err := w.accountTree.GetProof(LastAccountIndex)
		if err != nil {
			return accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore, err
		}
		// set account merkle proof
		merkleProofsAccountBefore[accountCount], err = SetFixedAccountArray(accountMerkleProofs)
		if err != nil {
			return accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore, err
		}
		for i := 0; i < NbAccountAssetsPerAccount; i++ {
			assetMerkleProof, err := emptyAssetTree.GetProof(0)
			if err != nil {
				return accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore, err
			}
			merkleProofsAccountAssetsBefore[accountCount][i], err = SetFixedAccountAssetArray(assetMerkleProof)
			if err != nil {
				return accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore, err
			}
		}
		accountCount++

	}
	return accountRootBefore, accountsInfoBefore, merkleProofsAccountAssetsBefore, merkleProofsAccountBefore, nil
}

func (w *WitnessHelper) constructNftWitness(
	proverNftInfo *NftWitnessInfo,
) (
	// nft root before
	nftRootBefore []byte,
	// nft before
	nftBefore *cryptoTypes.Nft,
	// before nft tree merkle proof
	merkleProofsNftBefore [NftMerkleLevels][]byte,
	err error,
) {
	nftRootBefore = w.nftTree.Root()
	if proverNftInfo == nil {
		nftMerkleProofs, err := w.nftTree.GetProof(LastNftIndex)
		if err != nil {
			return nftRootBefore, nftBefore, merkleProofsNftBefore, err
		}
		merkleProofsNftBefore, err = SetFixedNftArray(nftMerkleProofs)
		if err != nil {
			return nftRootBefore, nftBefore, merkleProofsNftBefore, err
		}
		nftBefore = cryptoTypes.EmptyNft(LastNftIndex)
		return nftRootBefore, nftBefore, merkleProofsNftBefore, nil
	}
	nftMerkleProofs, err := w.nftTree.GetProof(uint64(proverNftInfo.NftInfo.NftIndex))
	if err != nil {
		return nftRootBefore, nftBefore, merkleProofsNftBefore, err
	}
	merkleProofsNftBefore, err = SetFixedNftArray(nftMerkleProofs)
	if err != nil {
		return nftRootBefore, nftBefore, merkleProofsNftBefore, err
	}
	nftL1TokenId, isValid := new(big.Int).SetString(proverNftInfo.NftInfo.NftL1TokenId, 10)
	if !isValid {
		return nftRootBefore, nftBefore, merkleProofsNftBefore, fmt.Errorf("unable to parse big int")
	}
	nftBefore = &cryptoTypes.Nft{
		NftIndex:            proverNftInfo.NftInfo.NftIndex,
		NftContentHash:      common.FromHex(proverNftInfo.NftInfo.NftContentHash),
		CreatorAccountIndex: proverNftInfo.NftInfo.CreatorAccountIndex,
		OwnerAccountIndex:   proverNftInfo.NftInfo.OwnerAccountIndex,
		NftL1Address:        new(big.Int).SetBytes(common.FromHex(proverNftInfo.NftInfo.NftL1Address)),
		NftL1TokenId:        nftL1TokenId,
		CreatorTreasuryRate: proverNftInfo.NftInfo.CreatorTreasuryRate,
		CollectionId:        proverNftInfo.NftInfo.CollectionId,
	}

	nBalance, err := chain.ComputeNewBalance(
		proverNftInfo.NftRelatedTxDetail.AssetType,
		proverNftInfo.NftRelatedTxDetail.Balance,
		proverNftInfo.NftRelatedTxDetail.BalanceDelta,
	)
	if err != nil {
		return nftRootBefore, nftBefore, merkleProofsNftBefore, err
	}
	nNftInfo, err := types.ParseNftInfo(nBalance)
	if err != nil {
		return nftRootBefore, nftBefore, merkleProofsNftBefore, err
	}
	nNftHash, err := tree.ComputeNftAssetLeafHash(
		nNftInfo.CreatorAccountIndex,
		nNftInfo.OwnerAccountIndex,
		nNftInfo.NftContentHash,
		nNftInfo.NftL1Address,
		nNftInfo.NftL1TokenId,
		nNftInfo.CreatorTreasuryRate,
		nNftInfo.CollectionId,
	)

	if err != nil {
		return nftRootBefore, nftBefore, merkleProofsNftBefore, err
	}
	err = w.nftTree.Set(uint64(proverNftInfo.NftInfo.NftIndex), nNftHash)
	if err != nil {
		return nftRootBefore, nftBefore, merkleProofsNftBefore, err
	}
	if err != nil {
		return nftRootBefore, nftBefore, merkleProofsNftBefore, err
	}
	return nftRootBefore, nftBefore, merkleProofsNftBefore, nil
}

func SetFixedAccountArray(proof [][]byte) (res [AccountMerkleLevels][]byte, err error) {
	if len(proof) != AccountMerkleLevels {
		return res, fmt.Errorf("invalid size")
	}
	copy(res[:], proof[:])
	return res, nil
}

func SetFixedAccountAssetArray(proof [][]byte) (res [AssetMerkleLevels][]byte, err error) {
	if len(proof) != AssetMerkleLevels {
		return res, fmt.Errorf("invalid size")
	}
	copy(res[:], proof[:])
	return res, nil
}

func SetFixedNftArray(proof [][]byte) (res [NftMerkleLevels][]byte, err error) {
	if len(proof) != NftMerkleLevels {
		return res, fmt.Errorf("invalid size")
	}
	copy(res[:], proof[:])
	return res, nil
}

func (w *WitnessHelper) constructSimpleWitnessInfo(oTx *tx.Tx) (
	accountKeys []int64,
	accountWitnessInfo []*AccountWitnessInfo,
	nftWitnessInfo *NftWitnessInfo,
	err error,
) {
	var (
		// dbinitializer account asset map, because if we have the same asset detail, the before will be the after of the last one
		accountAssetMap  = make(map[int64]map[int64]*types.AccountAsset)
		accountMap       = make(map[int64]*account.Account)
		lastAccountOrder = int64(-2)
		accountCount     = -1
	)
	// dbinitializer prover account map
	if oTx.TxType == types.TxTypeRegisterZns {
		accountKeys = append(accountKeys, oTx.AccountIndex)
	}
	for _, txDetail := range oTx.TxDetails {
		// if tx detail is from gas account
		if txDetail.IsGas {
			continue
		}
		switch txDetail.AssetType {
		case types.FungibleAssetType:
			// get account info
			if accountMap[txDetail.AccountIndex] == nil {
				accountInfo, err := w.accountModel.GetConfirmedAccountByIndex(txDetail.AccountIndex)
				if err != nil {
					return nil, nil, nil, err
				}
				// get current nonce
				accountInfo.Nonce = txDetail.Nonce
				accountMap[txDetail.AccountIndex] = accountInfo
			} else {
				if lastAccountOrder != txDetail.AccountOrder {
					if oTx.AccountIndex == txDetail.AccountIndex {
						accountMap[txDetail.AccountIndex].Nonce = oTx.Nonce + 1
					}
				}
			}
			if lastAccountOrder != txDetail.AccountOrder {
				accountKeys = append(accountKeys, txDetail.AccountIndex)
				lastAccountOrder = txDetail.AccountOrder
				accountWitnessInfo = append(accountWitnessInfo, &AccountWitnessInfo{
					AccountInfo: &account.Account{
						AccountIndex:    accountMap[txDetail.AccountIndex].AccountIndex,
						AccountName:     accountMap[txDetail.AccountIndex].AccountName,
						PublicKey:       accountMap[txDetail.AccountIndex].PublicKey,
						AccountNameHash: accountMap[txDetail.AccountIndex].AccountNameHash,
						L1Address:       accountMap[txDetail.AccountIndex].L1Address,
						Nonce:           accountMap[txDetail.AccountIndex].Nonce,
						CollectionNonce: txDetail.CollectionNonce,
						AssetInfo:       accountMap[txDetail.AccountIndex].AssetInfo,
						AssetRoot:       accountMap[txDetail.AccountIndex].AssetRoot,
						Status:          accountMap[txDetail.AccountIndex].Status,
					},
				})
				accountCount++
			}
			if accountAssetMap[txDetail.AccountIndex] == nil {
				accountAssetMap[txDetail.AccountIndex] = make(map[int64]*types.AccountAsset)
			}
			if accountAssetMap[txDetail.AccountIndex][txDetail.AssetId] == nil {
				// set account before info
				oAsset, err := types.ParseAccountAsset(txDetail.Balance)
				if err != nil {
					return nil, nil, nil, err
				}
				accountWitnessInfo[accountCount].AccountAssets = append(
					accountWitnessInfo[accountCount].AccountAssets,
					oAsset,
				)
			} else {
				// set account before info
				accountWitnessInfo[accountCount].AccountAssets = append(
					accountWitnessInfo[accountCount].AccountAssets,
					&types.AccountAsset{
						AssetId:                  accountAssetMap[txDetail.AccountIndex][txDetail.AssetId].AssetId,
						Balance:                  accountAssetMap[txDetail.AccountIndex][txDetail.AssetId].Balance,
						OfferCanceledOrFinalized: accountAssetMap[txDetail.AccountIndex][txDetail.AssetId].OfferCanceledOrFinalized,
					},
				)
			}
			// set tx detail
			accountWitnessInfo[accountCount].AssetsRelatedTxDetails = append(
				accountWitnessInfo[accountCount].AssetsRelatedTxDetails,
				txDetail,
			)
			// update asset info
			newBalance, err := chain.ComputeNewBalance(txDetail.AssetType, txDetail.Balance, txDetail.BalanceDelta)
			if err != nil {
				return nil, nil, nil, err
			}
			nAsset, err := types.ParseAccountAsset(newBalance)
			if err != nil {
				return nil, nil, nil, err
			}
			accountAssetMap[txDetail.AccountIndex][txDetail.AssetId] = nAsset
		case types.NftAssetType:
			nftWitnessInfo = new(NftWitnessInfo)
			nftWitnessInfo.NftRelatedTxDetail = txDetail
			nftInfo, err := types.ParseNftInfo(txDetail.Balance)
			if err != nil {
				return nil, nil, nil, err
			}
			nftWitnessInfo.NftInfo = nftInfo
		case types.CollectionNonceAssetType:
			// get account info
			if accountMap[txDetail.AccountIndex] == nil {
				accountInfo, err := w.accountModel.GetConfirmedAccountByIndex(txDetail.AccountIndex)
				if err != nil {
					return nil, nil, nil, err
				}
				// get current nonce
				accountInfo.Nonce = txDetail.Nonce
				accountInfo.CollectionNonce = txDetail.CollectionNonce
				accountMap[txDetail.AccountIndex] = accountInfo
				if lastAccountOrder != txDetail.AccountOrder {
					accountKeys = append(accountKeys, txDetail.AccountIndex)
					lastAccountOrder = txDetail.AccountOrder
					accountWitnessInfo = append(accountWitnessInfo, &AccountWitnessInfo{
						AccountInfo: &account.Account{
							AccountIndex:    accountMap[txDetail.AccountIndex].AccountIndex,
							AccountName:     accountMap[txDetail.AccountIndex].AccountName,
							PublicKey:       accountMap[txDetail.AccountIndex].PublicKey,
							AccountNameHash: accountMap[txDetail.AccountIndex].AccountNameHash,
							L1Address:       accountMap[txDetail.AccountIndex].L1Address,
							Nonce:           accountMap[txDetail.AccountIndex].Nonce,
							CollectionNonce: txDetail.CollectionNonce,
							AssetInfo:       accountMap[txDetail.AccountIndex].AssetInfo,
							AssetRoot:       accountMap[txDetail.AccountIndex].AssetRoot,
							Status:          accountMap[txDetail.AccountIndex].Status,
						},
					})
					accountCount++
				}
			} else {
				accountMap[txDetail.AccountIndex].Nonce = txDetail.Nonce
				accountMap[txDetail.AccountIndex].CollectionNonce = txDetail.CollectionNonce
			}
		default:
			return nil, nil, nil,
				fmt.Errorf("invalid asset type")
		}
	}
	return accountKeys, accountWitnessInfo, nftWitnessInfo, nil
}

func (w *WitnessHelper) ConstructGasWitness(block *block.Block) (cryptoGas *GasWitness, err error) {
	var gas *circuit.Gas

	needGas := false
	gasChanges := make(map[int64]*big.Int)
	for _, assetId := range types.GasAssets {
		gasChanges[assetId] = types.ZeroBigInt
	}
	for _, tx := range block.Txs {
		if types.IsL2Tx(tx.TxType) {
			needGas = true
			for _, txDetail := range tx.TxDetails {
				if txDetail.IsGas {
					assetDelta, err := types.ParseAccountAsset(txDetail.BalanceDelta)
					if err != nil {
						return nil, err
					}
					gasChanges[assetDelta.AssetId] = ffmath.Add(gasChanges[assetDelta.AssetId], assetDelta.Balance)
				}
			}
		}
	}

	gasAccountIndex := types.GasAccount
	emptyAssetTree, err := tree.NewMemAccountAssetTree()
	if err != nil {
		return nil, err
	}
	if !needGas { // no need of gas for this block
		accountInfoBefore := cryptoTypes.EmptyGasAccount(gasAccountIndex, tree.NilAccountAssetRoot)
		accountMerkleProofs, err := w.accountTree.GetProof(uint64(gasAccountIndex))
		if err != nil {
			return nil, err
		}
		merkleProofsAccountBefore, err := SetFixedAccountArray(accountMerkleProofs)
		if err != nil {
			return nil, err
		}
		merkleProofsAccountAssetsBefore := make([][AssetMerkleLevels][]byte, 0)
		for _, assetId := range types.GasAssets {
			accountInfoBefore.AssetsInfo = append(accountInfoBefore.AssetsInfo, cryptoTypes.EmptyAccountAsset(assetId))
			assetMerkleProof, err := emptyAssetTree.GetProof(uint64(assetId))
			if err != nil {
				return nil, err
			}
			merkleProofsAccountAssetBefore, err := SetFixedAccountAssetArray(assetMerkleProof)
			if err != nil {
				return nil, err
			}
			merkleProofsAccountAssetsBefore = append(merkleProofsAccountAssetsBefore, merkleProofsAccountAssetBefore)
		}
		gas = &circuit.Gas{
			GasAssetCount:                   len(types.GasAssets),
			AccountInfoBefore:               accountInfoBefore,
			MerkleProofsAccountBefore:       merkleProofsAccountBefore,
			MerkleProofsAccountAssetsBefore: merkleProofsAccountAssetsBefore,
		}
	} else {
		pk, err := common2.ParsePubKey(w.gasAccountInfo.PublicKey)
		if err != nil {
			return nil, err
		}
		accountInfoBefore := &cryptoTypes.GasAccount{
			AccountIndex:    gasAccountIndex,
			AccountNameHash: common.FromHex(w.gasAccountInfo.AccountNameHash),
			AccountPk:       pk,
			Nonce:           w.gasAccountInfo.Nonce,
			CollectionNonce: w.gasAccountInfo.CollectionNonce,
			AssetRoot:       w.assetTrees.Get(gasAccountIndex).Root(),
		}
		logx.Infof("old gas account asset root: %s", common.Bytes2Hex(w.assetTrees.Get(gasAccountIndex).Root()))

		accountMerkleProofs, err := w.accountTree.GetProof(uint64(gasAccountIndex))
		if err != nil {
			return nil, err
		}
		merkleProofsAccountBefore, err := SetFixedAccountArray(accountMerkleProofs)
		if err != nil {
			return nil, err
		}
		merkleProofsAccountAssetsBefore := make([][AssetMerkleLevels][]byte, 0)
		for _, assetId := range types.GasAssets {
			assetMerkleProof, err := w.assetTrees.Get(gasAccountIndex).GetProof(uint64(assetId))
			if err != nil {
				return nil, err
			}
			balanceBefore := types.ZeroBigInt
			offerCanceledOrFinalized := types.ZeroBigInt
			if asset, ok := w.gasAccountInfo.AssetInfo[assetId]; ok {
				balanceBefore = asset.Balance
				offerCanceledOrFinalized = asset.OfferCanceledOrFinalized
			}
			accountInfoBefore.AssetsInfo = append(accountInfoBefore.AssetsInfo, &cryptoTypes.AccountAsset{
				AssetId:                  assetId,
				Balance:                  balanceBefore,
				OfferCanceledOrFinalized: offerCanceledOrFinalized,
			})

			// set merkle proof
			merkleProofsAccountAssetBefore, err := SetFixedAccountAssetArray(assetMerkleProof)
			if err != nil {
				return nil, err
			}
			merkleProofsAccountAssetsBefore = append(merkleProofsAccountAssetsBefore, merkleProofsAccountAssetBefore)

			balanceAfter := ffmath.Add(balanceBefore, gasChanges[assetId])
			nAssetHash, err := tree.ComputeAccountAssetLeafHash(balanceAfter.String(), offerCanceledOrFinalized.String())
			if err != nil {
				return nil, err
			}
			err = w.assetTrees.Get(gasAccountIndex).Set(uint64(assetId), nAssetHash)
			if err != nil {
				return nil, err
			}

		}
		logx.Infof("new gas account asset root: %s", common.Bytes2Hex(w.assetTrees.Get(gasAccountIndex).Root()))

		nAccountHash, err := tree.ComputeAccountLeafHash(
			w.gasAccountInfo.AccountNameHash,
			w.gasAccountInfo.PublicKey,
			w.gasAccountInfo.Nonce,
			w.gasAccountInfo.CollectionNonce,
			w.assetTrees.Get(gasAccountIndex).Root(),
		)
		if err != nil {
			return nil, err
		}
		err = w.accountTree.Set(uint64(gasAccountIndex), nAccountHash)
		if err != nil {
			return nil, err
		}
		gas = &circuit.Gas{
			GasAssetCount:                   len(types.GasAssets),
			AccountInfoBefore:               accountInfoBefore,
			MerkleProofsAccountBefore:       merkleProofsAccountBefore,
			MerkleProofsAccountAssetsBefore: merkleProofsAccountAssetsBefore,
		}
	}

	return gas, nil
}

func (w *WitnessHelper) ResetCache(height int64) error {
	w.gasAccountInfo = nil
	history, err := w.accountHistoryModel.GetLatestAccountHistory(types.GasAccount, height)
	if err != nil && err != types.DbErrNotFound {
		return err
	}

	if history != nil {
		gasAccount, err := w.accountModel.GetConfirmedAccountByIndex(types.GasAccount)
		if err != nil && err != types.DbErrNotFound {
			return err
		}
		gasAccount.Nonce = history.Nonce
		gasAccount.CollectionNonce = history.CollectionNonce
		gasAccount.AssetInfo = history.AssetInfo
		gasAccount.AssetRoot = history.AssetRoot
		formatGasAccount, err := chain.ToFormatAccountInfo(gasAccount)
		if err != nil {
			return err
		}

		w.gasAccountInfo = formatGasAccount
	}
	return nil
}
