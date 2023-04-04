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
	"math/big"
)

const (
	FungibleAssetType        = 1
	NftAssetType             = 2
	CollectionNonceAssetType = 3
	ChangePubKeyType         = 4
	CreateAccountType        = 5

	BuyOfferType  = 0
	SellOfferType = 1
)

type AccountAsset struct {
	AssetId                  int64
	Balance                  *big.Int
	OfferCanceledOrFinalized *big.Int
}

func (asset *AccountAsset) DeepCopy() *AccountAsset {
	return &AccountAsset{
		AssetId:                  asset.AssetId,
		Balance:                  big.NewInt(0).Set(asset.Balance),
		OfferCanceledOrFinalized: big.NewInt(0).Set(asset.OfferCanceledOrFinalized),
	}
}

func ConstructAccountAsset(assetId int64, balance *big.Int, offerCanceledOrFinalized *big.Int) *AccountAsset {
	return &AccountAsset{
		assetId,
		balance,
		offerCanceledOrFinalized,
	}
}

func ParseAccountAsset(balance string) (asset *AccountAsset, err error) {
	err = json.Unmarshal([]byte(balance), &asset)
	if err != nil {
		return nil, JsonErrUnmarshal
	}
	return asset, nil
}

func (asset *AccountAsset) String() (info string) {
	infoBytes, _ := json.Marshal(asset)
	return string(infoBytes)
}

type AccountInfo struct {
	AccountId       int64
	AccountIndex    int64
	PublicKey       string
	L1Address       string
	Nonce           int64
	CollectionNonce int64
	AssetInfo       map[int64]*AccountAsset // key: index, value: balance
	AssetRoot       string
	Status          int
}

func (ai *AccountInfo) DeepCopy() *AccountInfo {
	assetInfo := make(map[int64]*AccountAsset)
	for assetId, asset := range ai.AssetInfo {
		assetInfo[assetId] = asset.DeepCopy()
	}

	newAccountInfo := &AccountInfo{
		AccountId:       ai.AccountId,
		AccountIndex:    ai.AccountIndex,
		PublicKey:       ai.PublicKey,
		L1Address:       ai.L1Address,
		Nonce:           ai.Nonce,
		CollectionNonce: ai.CollectionNonce,
		AssetInfo:       assetInfo,
		AssetRoot:       ai.AssetRoot,
		Status:          ai.Status,
	}
	return newAccountInfo
}
