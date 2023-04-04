package query

import (
	bsmt "github.com/bnb-chain/zkbnb-smt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"
	"strconv"

	"github.com/bnb-chain/zkbnb/tools/query/internal/config"
	"github.com/bnb-chain/zkbnb/tools/query/internal/svc"
	"github.com/bnb-chain/zkbnb/tree"
)

func QueryTreeDB(
	configFile string,
	blockHeight int64,
	serviceName string,
	batchSize int,
	fromHistory bool,
) {
	var c config.Config
	conf.MustLoad(configFile, &c)
	ctx := svc.NewServiceContext(c)
	logx.MustSetup(c.LogConf)
	logx.DisableStat()
	proc.AddShutdownListener(func() {
		logx.Close()
	})

	treeCtx, err := tree.NewContext(serviceName, c.TreeDB.Driver, false, true, c.TreeDB.RoutinePoolSize, &c.TreeDB.LevelDBOption, &c.TreeDB.RedisDBOption)
	if err != nil {
		logx.Errorf("Init tree database failed: %s", err)
		return
	}

	treeCtx.SetOptions(bsmt.InitializeVersion(0))
	treeCtx.SetBatchReloadSize(batchSize)
	err = tree.SetupTreeDB(treeCtx)
	if err != nil {
		logx.Errorf("Init tree database failed: %s", err)
		return
	}

	// dbinitializer accountTree and accountStateTrees
	accountTree, accountAssetTrees, err := tree.InitAccountTree(
		ctx.AccountModel,
		ctx.AccountHistoryModel,
		make([]int64, 0),
		blockHeight,
		treeCtx,
		c.TreeDB.AssetTreeCacheSize,
		fromHistory,
	)
	if err != nil {
		logx.Error("InitMerkleTree error:", err)
		return
	}
	if len(ctx.Config.AccountIndexes) > 0 {
		for _, accountIndex := range ctx.Config.AccountIndexes {
			assetRoot := common.Bytes2Hex(accountAssetTrees.Get(accountIndex).Root())
			logx.Infof("asset tree root accountIndex=%s,assetRoot=%s,versions=%s,latestVersion=%s", strconv.FormatInt(accountIndex, 10), assetRoot,
				formatVersion(accountAssetTrees.Get(accountIndex).Versions()), strconv.FormatUint(uint64(accountAssetTrees.Get(accountIndex).LatestVersion()), 10))
			for i := 0; i < 20; i++ {
				assetOne, err := accountAssetTrees.Get(accountIndex).Get(uint64(i), nil)
				if err != nil {
					continue
				}
				logx.Infof("asset tree accountIndex=%s,assetId=%s,assetRoot=%s", strconv.FormatInt(accountIndex, 10), strconv.FormatInt(int64(i), 10), common.Bytes2Hex(assetOne))
			}
			//accountAssetTrees.Get(accountIndex).PrintLeaves()
		}
	}
	stateRoot := common.Bytes2Hex(accountTree.Root())
	logx.Infof("account tree accountRoot=%s,versions=%s,,latestVersion=%s", stateRoot, formatVersion(accountTree.Versions()), strconv.FormatUint(uint64(accountTree.LatestVersion()), 10))
	// dbinitializer nftTree
	nftTree, err := tree.InitNftTree(
		ctx.NftModel,
		ctx.NftHistoryModel,
		blockHeight,
		treeCtx, fromHistory)
	if err != nil {
		logx.Errorf("InitNftTree error: %s", err.Error())
		return
	}
	nftRoot := common.Bytes2Hex(nftTree.Root())
	logx.Infof("nft tree nftRoot=%s,versions=%s,,latestVersion=%s", nftRoot, formatVersion(nftTree.Versions()), strconv.FormatUint(uint64(nftTree.LatestVersion()), 10))
}

func formatVersion(versions []bsmt.Version) string {
	str := "["
	for _, version := range versions {
		str += strconv.FormatUint(uint64(version), 10) + ","
	}
	if str != "[" {
		str = str[0 : len(str)-1]
	}
	str += "]"

	return str
}
