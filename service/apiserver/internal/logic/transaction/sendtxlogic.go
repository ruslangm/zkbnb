package transaction

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/bnb-chain/zkbnb/core"
	"github.com/bnb-chain/zkbnb/dao/tx"
	"github.com/bnb-chain/zkbnb/service/apiserver/internal/svc"
	"github.com/bnb-chain/zkbnb/service/apiserver/internal/types"
	types2 "github.com/bnb-chain/zkbnb/types"
)

type SendTxLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSendTxLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendTxLogic {
	return &SendTxLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (s *SendTxLogic) SendTx(req *types.ReqSendTx) (resp *types.TxHash, err error) {
	txStatuses := []int64{tx.StatusPending}
	pendingTxCount, err := s.svcCtx.TxPoolModel.GetTxsTotalCount(tx.GetTxWithStatuses(txStatuses))
	if err != nil {
		return nil, types2.AppErrInternal
	}

	if s.svcCtx.Config.TxPool.MaxPendingTxCount > 0 && pendingTxCount >= int64(s.svcCtx.Config.TxPool.MaxPendingTxCount) {
		return nil, types2.AppErrTooManyTxs
	}

	resp = &types.TxHash{}
	bc, err := core.NewBlockChainForDryRun(s.svcCtx.AccountModel, s.svcCtx.NftModel, s.svcCtx.TxPoolModel,
		s.svcCtx.AssetModel, s.svcCtx.SysConfigModel, s.svcCtx.RedisCache)
	if err != nil {
		logx.Error("fail to init blockchain runner:", err)
		return nil, types2.AppErrInternal
	}
	newTx := &tx.Tx{
		TxHash: types2.EmptyTxHash, // Would be computed in prepare method of executors.
		TxType: int64(req.TxType),
		TxInfo: req.TxInfo,

		GasFeeAssetId: types2.NilAssetId,
		GasFee:        types2.NilAssetAmount,
		NftIndex:      types2.NilNftIndex,
		CollectionId:  types2.NilCollectionNonce,
		AssetId:       types2.NilAssetId,
		TxAmount:      types2.NilAssetAmount,
		NativeAddress: types2.EmptyL1Address,

		BlockHeight: types2.NilBlockHeight,
		TxStatus:    tx.StatusPending,
	}

	err = bc.ApplyTransaction(newTx)
	if err != nil {
		return resp, err
	}
	if err := s.svcCtx.TxPoolModel.CreateTxs([]*tx.Tx{newTx}); err != nil {
		logx.Errorf("fail to create pool tx: %v, err: %s", newTx, err.Error())
		return resp, types2.AppErrInternal
	}

	resp.TxHash = newTx.TxHash
	return resp, nil
}
