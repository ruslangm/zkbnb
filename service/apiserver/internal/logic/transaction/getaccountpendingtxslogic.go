package transaction

import (
	"context"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/bnb-chain/zkbnb/dao/tx"
	"github.com/bnb-chain/zkbnb/service/apiserver/internal/logic/utils"
	"github.com/bnb-chain/zkbnb/service/apiserver/internal/svc"
	"github.com/bnb-chain/zkbnb/service/apiserver/internal/types"
	types2 "github.com/bnb-chain/zkbnb/types"
)

type GetAccountPendingTxsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAccountPendingTxsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAccountPendingTxsLogic {
	return &GetAccountPendingTxsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAccountPendingTxsLogic) GetAccountPendingTxs(req *types.ReqGetAccountPendingTxs) (resp *types.Txs, err error) {
	resp = &types.Txs{
		Txs: make([]*types.Tx, 0),
	}

	accountIndex := int64(0)
	switch req.By {
	case queryByAccountIndex:
		accountIndex, err = strconv.ParseInt(req.Value, 10, 64)
		if err != nil || accountIndex < 0 {
			return nil, types2.AppErrInvalidAccountIndex
		}
	case queryByL1Address:
		accountIndex, err = l.svcCtx.MemCache.GetAccountIndexByL1Address(req.Value)
	default:
		return nil, types2.AppErrInvalidParam.RefineError("param by should be account_index|l1_address")
	}

	if err != nil {
		if err == types2.DbErrNotFound {
			return resp, nil
		}
		return nil, types2.AppErrInternal
	}

	options := []tx.GetTxOptionFunc{}
	if len(req.Types) > 0 {
		options = append(options, tx.GetTxWithTypes(req.Types))
	}

	poolTxs, err := l.svcCtx.TxPoolModel.GetPendingTxsByAccountIndex(accountIndex, options...)
	if err != nil {
		if err != types2.DbErrNotFound {
			return nil, types2.AppErrInternal
		}
	}

	resp.Total = uint32(len(poolTxs))
	for _, poolTx := range poolTxs {
		tx := utils.ConvertTx(poolTx)
		tx.L1Address, _ = l.svcCtx.MemCache.GetL1AddressByIndex(tx.AccountIndex)
		tx.AssetName, _ = l.svcCtx.MemCache.GetAssetNameById(tx.AssetId)
		if tx.ToAccountIndex >= 0 {
			tx.ToL1Address, _ = l.svcCtx.MemCache.GetL1AddressByIndex(tx.ToAccountIndex)
		}
		resp.Txs = append(resp.Txs, tx)
	}
	return resp, nil
}
