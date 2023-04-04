package transaction

import (
	"context"
	"github.com/bnb-chain/zkbnb/dao/tx"
	"github.com/bnb-chain/zkbnb/service/apiserver/internal/logic/utils"
	types2 "github.com/bnb-chain/zkbnb/types"
	"strconv"

	"github.com/bnb-chain/zkbnb/service/apiserver/internal/svc"
	"github.com/bnb-chain/zkbnb/service/apiserver/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMergedAccountTxsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMergedAccountTxsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMergedAccountTxsLogic {
	return &GetMergedAccountTxsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMergedAccountTxsLogic) GetMergedAccountTxs(req *types.ReqGetAccountTxs) (resp *types.Txs, err error) {

	resp = &types.Txs{
		Txs: make([]*types.Tx, 0, req.Limit),
	}

	accountIndex, err := l.fetchAccountIndexFromReq(req)
	if err != nil {
		if err == types2.DbErrNotFound {
			return resp, nil
		}
		return nil, err
	}

	var options []tx.GetTxOptionFunc
	if len(req.Types) > 0 {
		options = append(options, tx.GetTxWithTypes(req.Types))
	}

	poolTxCount, err := l.svcCtx.TxPoolModel.GetTxsCountByAccountIndex(accountIndex, options...)
	if err != nil {
		return nil, types2.DbErrSqlOperation
	}
	txCount, err := l.svcCtx.TxModel.GetTxsCountByAccountIndex(accountIndex, options...)
	if err != nil {
		return nil, types2.DbErrSqlOperation
	}
	replicateTxCount, err := l.svcCtx.TxModel.GetReplicateTxsCountByAccountIndex(accountIndex, options...)
	if err != nil {
		return nil, types2.DbErrSqlOperation
	}

	totalTxCount := poolTxCount + txCount - replicateTxCount
	if totalTxCount == 0 || totalTxCount <= int64(req.Offset) {
		return resp, nil
	}

	txsList := make([]*types.Tx, 0, req.Limit)

	if (poolTxCount - replicateTxCount) < int64(req.Offset) {
		txOffset := int64(req.Offset) - poolTxCount + replicateTxCount
		txs, err := l.svcCtx.TxModel.GetTxsByAccountIndex(accountIndex, int64(req.Limit), txOffset, options...)
		if err != nil && err != types2.DbErrNotFound {
			return nil, types2.DbErrSqlOperation
		}
		txsList = l.appendTxsList(txsList, txs)
	} else {
		poolTxLimit := poolTxCount - int64(req.Offset) - replicateTxCount
		if poolTxLimit > int64(req.Limit) {
			poolTxs, err := l.svcCtx.TxPoolModel.GetTxsByAccountIndex(accountIndex, int64(req.Limit), int64(req.Offset), options...)
			if err != nil && err != types2.DbErrNotFound {
				return nil, types2.DbErrSqlOperation
			}
			txsList = l.appendTxsList(txsList, poolTxs)
		} else {
			poolTxs, err := l.svcCtx.TxPoolModel.GetTxsByAccountIndex(accountIndex, poolTxLimit, int64(req.Offset), options...)
			if err != nil && err != types2.DbErrNotFound {
				return nil, types2.DbErrSqlOperation
			}
			txsList = l.appendTxsList(txsList, poolTxs)
			txLimit := int64(req.Limit) - poolTxLimit
			if txLimit > 0 {
				txs, err := l.svcCtx.TxModel.GetTxsByAccountIndex(accountIndex, txLimit, 0, options...)
				if err != nil && err != types2.DbErrNotFound {
					return nil, types2.DbErrSqlOperation
				}
				txsList = l.appendTxsList(txsList, txs)
			}
		}
	}

	resp = &types.Txs{
		Txs:   txsList,
		Total: uint32(totalTxCount),
	}
	return resp, nil
}

func (l *GetMergedAccountTxsLogic) fetchAccountIndexFromReq(req *types.ReqGetAccountTxs) (int64, error) {
	switch req.By {
	case queryByAccountIndex:
		accountIndex, err := strconv.ParseInt(req.Value, 10, 64)
		if err != nil || accountIndex < 0 {
			return accountIndex, types2.AppErrInvalidAccountIndex
		}
		return accountIndex, err
	case queryByL1Address:
		accountIndex, err := l.svcCtx.MemCache.GetAccountIndexByL1Address(req.Value)
		return accountIndex, err
	}
	return 0, types2.AppErrInvalidParam.RefineError("param by should be account_index|l1_address")
}

func (l *GetMergedAccountTxsLogic) appendTxsList(txsResultList []*types.Tx, txList []*tx.Tx) []*types.Tx {

	for _, dbTx := range txList {
		tx := utils.ConvertTx(dbTx)
		tx.L1Address, _ = l.svcCtx.MemCache.GetL1AddressByIndex(tx.AccountIndex)
		tx.AssetName, _ = l.svcCtx.MemCache.GetAssetNameById(tx.AssetId)
		if tx.ToAccountIndex >= 0 {
			tx.ToL1Address, _ = l.svcCtx.MemCache.GetL1AddressByIndex(tx.ToAccountIndex)
		}
		txsResultList = append(txsResultList, tx)
	}
	return txsResultList
}
