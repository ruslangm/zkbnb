package transaction

import (
	"context"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"

	blockdao "github.com/bnb-chain/zkbnb/dao/block"
	"github.com/bnb-chain/zkbnb/service/apiserver/internal/logic/utils"
	"github.com/bnb-chain/zkbnb/service/apiserver/internal/svc"
	"github.com/bnb-chain/zkbnb/service/apiserver/internal/types"
	types2 "github.com/bnb-chain/zkbnb/types"
)

const (
	queryByBlockHeight     = "block_height"
	queryByBlockCommitment = "block_commitment"
)

type GetBlockTxsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetBlockTxsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBlockTxsLogic {
	return &GetBlockTxsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetBlockTxsLogic) GetBlockTxs(req *types.ReqGetBlockTxs) (resp *types.Txs, err error) {
	resp = &types.Txs{
		Txs: make([]*types.Tx, 0),
	}

	var block *blockdao.Block
	switch req.By {
	case queryByBlockHeight:
		height := int64(0)
		height, err = strconv.ParseInt(req.Value, 10, 64)
		if err != nil || height < 0 {
			return nil, types2.AppErrInvalidBlockHeight
		}
		block, err = l.svcCtx.MemCache.GetBlockByHeightWithFallback(height, func() (interface{}, error) {
			return l.svcCtx.BlockModel.GetBlockByHeight(height)
		})
	case queryByBlockCommitment:
		block, err = l.svcCtx.MemCache.GetBlockByCommitmentWithFallback(req.Value, func() (interface{}, error) {
			return l.svcCtx.BlockModel.GetBlockByCommitment(req.Value)
		})
	default:
		return nil, types2.AppErrInvalidParam.RefineError("param by should be height|commitment")
	}

	if err != nil {
		if err == types2.DbErrNotFound {
			return resp, nil
		}
		return nil, types2.AppErrInternal
	}

	resp.Total = uint32(len(block.Txs))
	for _, dbTx := range block.Txs {
		tx := utils.ConvertTx(dbTx)
		tx.L1Address, _ = l.svcCtx.MemCache.GetL1AddressByIndex(tx.AccountIndex)
		tx.AssetName, _ = l.svcCtx.MemCache.GetAssetNameById(tx.AssetId)
		if tx.ToAccountIndex >= 0 {
			tx.ToL1Address, _ = l.svcCtx.MemCache.GetL1AddressByIndex(tx.ToAccountIndex)
		}
		resp.Txs = append(resp.Txs, tx)
	}
	return resp, nil
}
