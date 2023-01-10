package nft

import (
	"context"
	types2 "github.com/bnb-chain/zkbnb/types"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/bnb-chain/zkbnb/service/apiserver/internal/svc"
	"github.com/bnb-chain/zkbnb/service/apiserver/internal/types"
)

type GetNftByTxHashLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetNftByTxHashLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetNftByTxHashLogic {
	return &GetNftByTxHashLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetNftByTxHashLogic) GetNftByTxHash(req *types.ReqGetNftIndex) (resp *types.NftIndex, err error) {
	tx, err := l.svcCtx.TxModel.GetTxByHash(req.TxHash)
	if err != nil {
		if err == types2.DbErrNotFound {
			return nil, types2.AppErrNftNotFound
		}
		return nil, types2.AppErrInternal
	}

	return &types.NftIndex{
		Index: tx.NftIndex,
	}, nil
}
