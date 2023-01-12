package nft

import (
	"context"
	"github.com/bnb-chain/zkbnb/dao/nft"
	types2 "github.com/bnb-chain/zkbnb/types"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/bnb-chain/zkbnb/service/apiserver/internal/svc"
	"github.com/bnb-chain/zkbnb/service/apiserver/internal/types"
)

type UpdateNftByIndexLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateNftByIndexLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateNftByIndexLogic {
	return &UpdateNftByIndexLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateNftByIndexLogic) UpdateNftByIndex(req *types.ReqUpdateNft) (*types.History, error) {
	l2Nft, err := l.svcCtx.NftModel.GetNft(req.NftIndex)
	if err != nil {
		if err == types2.DbErrNotFound {
			return nil, types2.AppErrNftNotFound
		}
		return nil, types2.AppErrInternal
	}
	history := &nft.L2NftMetadataHistory{
		NftIndex: req.NftIndex,
		IpnsName: l2Nft.IpnsName,
		IpnsId:   l2Nft.IpnsId,
		Mutable:  req.Mutable,
		Status:   nft.NotConfirmed,
	}
	err = l.svcCtx.NftMetadataHistoryModel.CreateL2NftMetadataHistoryInTransact(history)
	if err != nil {
		return nil, err
	}
	return &types.History{
		Mutable: req.Mutable,
	}, nil
}
