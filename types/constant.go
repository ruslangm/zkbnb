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
	"math"
	"math/big"
)

const (
	EmptyNonce               = 0
	EmptyCollectionNonce     = 0
	EmptyAccountAssetInfo    = "{}"
	EmptyAccountIndex        = int64(0)
	EmptyNftContentHash      = "0"
	EmptyAccountNameHash     = "0"
	EmptyTxHash              = "0"
	EmptyL1Address           = "0"
	EmptyCreatorTreasuryRate = 0

	NilAccountName     = ""
	NilNftIndex        = int64(-1)
	NilAccountIndex    = int64(-1)
	NilBlockHeight     = -1
	NilAssetId         = -1
	NilAccountOrder    = -1
	NilNonce           = -1
	NilCollectionNonce = -1
	NilExpiredAt       = math.MaxInt64
	NilAssetAmount     = "0"

	GasAccount  = int64(1)
	BNBAssetId  = 0
	BUSDAssetId = 1
)

var (
	EmptyOfferCanceledOrFinalized = big.NewInt(0)
	GasAssets                     = [2]int64{0, 1}
)
