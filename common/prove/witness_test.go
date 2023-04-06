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
	"encoding/json"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/bnb-chain/zkbnb-crypto/circuit"
	"github.com/bnb-chain/zkbnb/dao/account"
	"github.com/bnb-chain/zkbnb/dao/block"
	"github.com/bnb-chain/zkbnb/dao/blockwitness"
	"github.com/bnb-chain/zkbnb/dao/nft"
	"github.com/bnb-chain/zkbnb/tree"
)

var (
	dsn                 = "host=localhost user=postgres password=ZkBNB@123 dbname=zkbnb port=5434 sslmode=disable"
	blockModel          block.BlockModel
	witnessModel        blockwitness.BlockWitnessModel
	accountModel        account.AccountModel
	accountHistoryModel account.AccountHistoryModel
	nftHistoryModel     nft.L2NftHistoryModel
	assetTreeCacheSize  = 512000
)

func TestConstructWitness(t *testing.T) {
	testDBSetup()
	defer testDBShutdown()
	maxTestBlockHeight := int64(49)
	for h := int64(1); h < maxTestBlockHeight; h++ {
		witnessHelper, err := getWitnessHelper(h - 1)
		assert.NoError(t, err)
		b, err := blockModel.GetPendingBlocksBetween(h, h)
		assert.NoError(t, err)
		w, err := witnessModel.GetBlockWitnessByHeight(h)
		assert.NoError(t, err)
		var cBlock circuit.Block
		err = json.Unmarshal([]byte(w.WitnessData), &cBlock)
		assert.NoError(t, err)
		err = witnessHelper.ResetCache(h)
		assert.NoError(t, err)
		for idx, tx := range b[0].Txs {
			txWitness, err := witnessHelper.ConstructTxWitness(tx, uint64(0))
			assert.NoError(t, err)
			expectedBz, _ := json.Marshal(cBlock.Txs[idx])
			actualBz, _ := json.Marshal(txWitness)
			assert.Equal(t, string(actualBz), string(expectedBz), fmt.Sprintf("block %d, tx %d generate witness failed, tx type: %d", h, idx, tx.TxType))
		}
		gasWitness, err := witnessHelper.ConstructGasWitness(b[0])
		assert.NoError(t, err)
		expectedBz, _ := json.Marshal(cBlock.Gas)
		actualBz, _ := json.Marshal(gasWitness)
		assert.Equal(t, string(actualBz), string(expectedBz), fmt.Sprintf("block %d, gas generate witness failed", h))
	}
}

func getWitnessHelper(blockHeight int64) (*WitnessHelper, error) {
	ctx, err := tree.NewContext("witness", tree.MemoryDB, false, false, 128, nil, nil)
	if err != nil {
		return nil, err
	}
	accountTree, accountAssetTrees, err := tree.InitAccountTree(accountModel, accountHistoryModel, make([]int64, 0), blockHeight, ctx, assetTreeCacheSize)
	if err != nil {
		return nil, err
	}
	nftTree, err := tree.InitNftTree(nftHistoryModel, blockHeight, ctx)
	if err != nil {
		return nil, err
	}
	return NewWitnessHelper(ctx,
		accountTree,
		nftTree,
		accountAssetTrees,
		accountModel,
		accountHistoryModel), nil
}

func testDBSetup() {
	testDBShutdown()
	time.Sleep(5 * time.Second)
	cmd := exec.Command("docker", "run", "--name", "postgres-ut-witness", "-p", "5434:5432",
		"-e", "POSTGRES_PASSWORD=ZkBNB@123", "-e", "POSTGRES_USER=postgres", "-e", "POSTGRES_DB=zkbnb",
		"-e", "PGDATA=/var/lib/postgresql/pgdata", "-d", "ghcr.io/bnb-chain/zkbnb/zkbnb-ut-postgres:latest")
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	time.Sleep(15 * time.Second)
	db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	blockModel = block.NewBlockModel(db)
	witnessModel = blockwitness.NewBlockWitnessModel(db)
	accountModel = account.NewAccountModel(db)
	accountHistoryModel = account.NewAccountHistoryModel(db)
	nftHistoryModel = nft.NewL2NftHistoryModel(db)
}

func testDBShutdown() {
	cmd := exec.Command("docker", "kill", "postgres-ut-witness")
	//nolint:errcheck
	cmd.Run()
	time.Sleep(time.Second)
	cmd = exec.Command("docker", "rm", "postgres-ut-witness")
	//nolint:errcheck
	cmd.Run()
}
